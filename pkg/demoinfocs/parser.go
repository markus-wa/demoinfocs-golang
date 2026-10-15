package demoinfocs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/golang/geo/r3"
	dp "github.com/markus-wa/godispatch"
	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v6/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/cstv"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/config"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// header contains information from a demo's header.
type header struct {
	Filestamp       string        // aka. File-type, must be HL2DEMO
	NetworkProtocol int           // Not sure what this is for
	ServerName      string        // Server's 'hostname' config value
	ClientName      string        // Usually 'GOTV Demo'
	MapName         string        // E.g. de_cache, de_nuke, cs_office, etc.
	GameDirectory   string        // Usually 'csgo'
	PlaybackTime    time.Duration // Demo duration in seconds (= PlaybackTicks / Server's tickrate)
	PlaybackTicks   int           // Game duration in ticks (= PlaybackTime * Server's tickrate)
	PlaybackFrames  int           // Amount of 'frames' aka demo-ticks recorded (= PlaybackTime * Demo's recording rate)
}

// FrameRate returns the frame rate of the demo (frames / demo-ticks per second).
// Not necessarily the tick-rate the server ran on during the game.
//
// Returns 0 if PlaybackTime or PlaybackFrames are 0 (corrupt demo headers).
func (h *header) FrameRate() float64 {
	if h.PlaybackTime == 0 {
		return 0
	}

	return float64(h.PlaybackFrames) / h.PlaybackTime.Seconds()
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
//
// Returns 0 if PlaybackTime or PlaybackFrames are 0 (corrupt demo headers).
func (h *header) FrameTime() time.Duration {
	if h.PlaybackFrames == 0 {
		return 0
	}

	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackFrames))
}

/*
Parser can parse a CS2 demo.
A Parser must be created with NewParser(); the zero value is not usable.

Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo
(the demo header is parsed automatically when parsing starts).

Use Parser.RegisterEventHandler() to receive notifications about events.

Example (without error handling):

	f, _ := os.Open("/path/to/demo.dem")
	p := demoinfocs.NewParser(f)
	defer p.Close()
	p.RegisterEventHandler(func(e events.BombExplode) {
		fmt.Printf("%c site went BOOM!\n", e.Site)
	})
	p.ParseToEnd(context.Background())

Prints out '{A/B} site went BOOM!' when a bomb explodes.

Errors of the parser itself (e.g. truncated or corrupt demos) are returned by ParseToEnd() and ParseNextFrame().
Panics of handlers are re-panicked on the calling goroutine as *PanicError, see ParseToEnd().
To stop parsing from a handler, cancel the context passed to ParseToEnd() - don't panic.

Code that depends on a Parser can be unit-tested by declaring a small interface with just the
methods it uses (e.g. RegisterEventHandler and ParseToEnd) and passing a fake in tests,
see examples/mocking.
*/
type Parser struct {
	// Important fields

	config                          config.Config
	demostream                      io.Reader      // the stream passed to NewParser(), opened lazily, see ensureOpen()
	bitReader                       *bit.BitReader // nil until the demo stream is opened, see ensureOpen()
	stParser                        *st.Parser
	additionalNetMessageCreators    map[int]config.NetMessageCreator // Map of net-message-IDs to NetMessageCreators (for parsing custom net-messages)
	msgQueue                        chan any                         // Queue of net-messages
	msgDispatcher                   *dp.Dispatcher                   // Net-message dispatcher
	decryptionKey                   []byte                           // Stored in `match730_*.dem.info` see MatchInfoDecryptionKey().
	gameEventHandler                gameEventHandler
	eventDispatcher                 *dp.Dispatcher
	currentFrame                    int     // Demo-frame, not ingame-tick
	readerFrame                     int     // Like currentFrame, but counted on the parsing goroutine, see queueFrameParsed()
	tickInterval                    float32 // Duration between ticks in seconds
	header                          *header // Pointer so we can check for nil
	gameState                       *GameState
	demoInfoProvider                demoInfoProvider // Provides demo infos to other packages that the core package depends on
	abort                           abortState       // Why parsing stops (errors, cancellation, handler panics), see aborted()
	done                            bool             // Set once parsing has finished (successfully or not), see finishParsing()
	doneErr                         error            // The result of parsing once done is set
	source2FallbackGameEventListBin []byte           // sv_hibernate_when_empty bug workaround
	ignorePacketEntitiesPanic       bool             // Report PacketEntities decoding failures as ParserWarn instead of failing (some POV demos have rare broken PacketEntities)
	/**
	 * Set to the client slot of the recording player.
	 * Always -1 for GOTV demos.
	 */
	recordingPlayerSlot           int
	disableMimicSource1GameEvents bool
	eagerVelocityTracking         bool
	snapshotTick                  int           // server tick of the last svc_PacketEntities, see Player.Velocity()
	processingSnapshot            bool          // true while a svc_PacketEntities is applied, see Player.Velocity()
	framePositionRecorders        []func() bool // see Player.Velocity()
	lazyVelocityTrackingWarned    bool

	// Additional fields, mainly caching & tracking things

	bombsiteA             bombsite
	bombsiteB             bombsite
	rawPlayers            map[int]*common.PlayerInfo                               // Maps entity IDs to 'raw' player info
	dirtyControllers      map[int]*dirtyPlayerController                           // Player controllers whose identity / connection state changed in the current entity packet
	provisionalNames      map[*common.Player]struct{}                              // Players whose name isn't from the controller yet (userinfo fallback or SteamID placeholder)
	triggers              map[int]*boundingBoxInformation                          // Maps entity IDs to triggers (used for bombsites)
	gameEventDescs        map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT // Maps game-event IDs to descriptors
	equipmentTypePerModel map[uint64]common.EquipmentType                          // Used to retrieve the EquipmentType of grenade projectiles based on models value. Source 2 only.
	stringTables          []*msg.CSVCMsg_CreateStringTable                         // Contains all created sendtables, needed when updating them
	delayedEventHandlers  []func()                                                 // Contains event handlers that need to be executed at the end of a tick (e.g. flash events because FlashDuration isn't updated before that)
	pendingMessagesCache  []pendingMessage                                         // Cache for pending messages that need to be dispatched after the current tick
	pendingMsgBufs        []*[]byte                                                // Pool entries backing pendingMessagesCache's buffers, returned to msgBufPool when the packet is done
	pendingMsgs           []proto.Message                                          // The decoded net-messages of the current packet, see decodeDemoPacket()
	snappyScratch         []byte                                                   // Reusable snappy decode destination for compressed frames (single-threaded frame parsing)
	userCmdStates         map[int32]*userCmdPlayerState                            // Per-player command-number ring reconstructed from full + delta updates
	userCmdButtonStates   map[int32]*userCmdButtonPlayerState                      // Per-player command-number ring containing buttonstate1 only
	hasUserCmdMessages    bool                                                     // True once a CSVCMsg_UserCommands message has been seen; replace the legacy m_nButtonDownMaskPrev prop
	lastUniqueID          uint64                                                   // Backs nextUniqueID; ids for grenades/infernos/equipment created during this parse
}

// nextUniqueID returns the next monotonically increasing id for entities created during this parse
// (grenade projectiles, infernos, equipment instances). Ids are unique within the Parser and
// reproducible across parses of the same demo (creation follows the deterministic parse order).
// Ids from different Parser instances may collide, so don't key data across demos by them.
//
// All construction happens on the message-handling goroutine, but the atomic keeps it
// race-detector-clean if that ever changes.
func (p *Parser) nextUniqueID() common.UniqueID {
	return common.UniqueID(atomic.AddUint64(&p.lastUniqueID, 1))
}

type bombsite struct {
	index  int
	center r3.Vector
}

type boundingBoxInformation struct {
	min r3.Vector
	max r3.Vector
}

func (bbi boundingBoxInformation) contains(point r3.Vector) bool {
	return point.X >= bbi.min.X && point.X <= bbi.max.X &&
		point.Y >= bbi.min.Y && point.Y <= bbi.max.Y &&
		point.Z >= bbi.min.Z && point.Z <= bbi.max.Z
}

// ServerClasses returns the server-classes of this demo.
// These are available after events.DataTablesParsed has been fired.
// Before that it is empty, but a value obtained early still reflects classes that are registered later.
func (p *Parser) ServerClasses() st.ServerClasses {
	return p.stParser.ServerClasses()
}

// GameState returns the current game-state.
// It contains most of the relevant information about the game such as players, teams, scores, grenades etc.
func (p *Parser) GameState() *GameState {
	return p.gameState
}

// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
// Starts with frame 0, should go up to header.PlaybackFrames but might not be the case (usually it's just close to it).
func (p *Parser) CurrentFrame() int {
	return p.currentFrame
}

// CurrentTime returns the time elapsed since the start of the demo
func (p *Parser) CurrentTime() time.Duration {
	return time.Duration(float32(p.gameState.ingameTick) * p.tickInterval * float32(time.Second))
}

// TickRate returns the tick-rate the server ran on during the game.
//
// Returns tick rate based on CSVCMsg_ServerInfo if possible.
// Otherwise returns tick rate based on demo header or -1 if the header info isn't available.
func (p *Parser) TickRate() float64 {
	if p.tickInterval != 0 {
		return 1.0 / float64(p.tickInterval)
	}

	if p.header != nil {
		return legacyTickRate(*p.header)
	}

	return -1
}

func legacyTickRate(h header) float64 {
	if h.PlaybackTime == 0 {
		return 0
	}

	return float64(h.PlaybackTicks) / h.PlaybackTime.Seconds()
}

// TickTime returns the time a single tick takes in seconds.
//
// Returns tick time based on CSVCMsg_ServerInfo if possible.
// Otherwise returns tick time based on demo header or -1 if the header info isn't available.
func (p *Parser) TickTime() time.Duration {
	if p.tickInterval != 0 {
		return time.Duration(float32(time.Second) * p.tickInterval)
	}

	if p.header != nil {
		return legayTickTime(*p.header)
	}

	return -1
}

func legayTickTime(h header) time.Duration {
	if h.PlaybackTicks == 0 {
		return 0
	}

	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackTicks))
}

// Progress returns the parsing progress from 0 to 1.
// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
//
// Might not be 100% correct since it's just based on the reported tick count of the header.
// May always return 0 if the demo header is corrupt.
func (p *Parser) Progress() float32 {
	if p.header == nil || p.header.PlaybackFrames == 0 {
		return 0
	}

	return float32(p.currentFrame) / float32(p.header.PlaybackFrames)
}

// FrameCount returns the total number of frames in the demo, or -1 if it is not yet known.
//
// For Source 2 (CS2) demos the total only becomes available once the trailing CDemoFileInfo has
// been parsed - typically at the very end of ParseToEnd - and is absent from the pre-parse header,
// so this returns -1 throughout parsing for those demos. Use it to branch cleanly rather than
// relying on Progress() (which returns 0 while the total is unknown).
func (p *Parser) FrameCount() int {
	if p.header == nil || p.header.PlaybackFrames <= 0 {
		return -1
	}

	return p.header.PlaybackFrames
}

/*
RegisterEventHandler registers a handler for game events.

The handler must be of type func(<EventType>) where EventType is the kind of event to be handled.
To catch all events func(any) can be used.

Example:

	parser.RegisterEventHandler(func(e events.WeaponFired) {
		fmt.Printf("%s fired his %s\n", e.Shooter.Name, e.Weapon.Type)
	})

Parameter handler has to be of type any because Go generics only work on functions, not methods.

Returns an identifier with which the handler can be removed via UnregisterEventHandler().
*/
func (p *Parser) RegisterEventHandler(handler any) dp.HandlerIdentifier {
	return p.eventDispatcher.RegisterHandler(handler) //nolint:forbidigo // public API, registers user handlers
}

// UnregisterEventHandler removes a game event handler via identifier.
//
// The identifier is returned at registration by RegisterEventHandler().
func (p *Parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.eventDispatcher.UnregisterHandler(identifier)
}

/*
RegisterNetMessageHandler registers a handler for net-messages.

The handler must be of type func(*<MessageType>) where MessageType is the kind of net-message to be handled.

Parameter handler has to be of type any because Go generics only work on functions, not methods.

Returns an identifier with which the handler can be removed via UnregisterNetMessageHandler().

See also: RegisterEventHandler()
*/
func (p *Parser) RegisterNetMessageHandler(handler any) dp.HandlerIdentifier {
	return p.msgDispatcher.RegisterHandler(handler) //nolint:forbidigo // public API, registers user handlers
}

// UnregisterNetMessageHandler removes a net-message handler via identifier.
//
// The identifier is returned at registration by RegisterNetMessageHandler().
func (p *Parser) UnregisterNetMessageHandler(identifier dp.HandlerIdentifier) {
	p.msgDispatcher.UnregisterHandler(identifier)
}

// Close closes any open resources used by the Parser (go routines, file handles).
// If the demo stream passed to NewParser() implements io.Closer, it's closed too.
// This must be called before discarding the Parser to avoid memory leaks.
// Returns an error if closing of underlying resources fails (all of them are closed regardless).
func (p *Parser) Close() error {
	p.msgDispatcher.RemoveAllQueues()

	var errs []error

	if p.bitReader != nil {
		// Doesn't close the demo stream, see demoReader.
		err := p.bitReader.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to close BitReader: %w", err))
		}
	}

	if closer, ok := p.demostream.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to close demo stream: %w", err))
		}
	}

	return errors.Join(errs...)
}

// ensureOpen opens the demo stream for reading, unless it's open already.
// It's called when parsing starts, so NewParser() doesn't read from the stream.
//
// May panic (e.g. with io.EOF for an empty stream), so it must be called within
// the recover scope of ParseToEnd() / ParseNextFrame().
func (p *Parser) ensureOpen() {
	if p.bitReader != nil {
		return
	}

	r := newDemoReader(p.demostream)

	if p.config.Format == config.FormatFile {
		p.bitReader = bit.NewLargeBitReader(r)
	} else {
		p.bitReader = bit.NewSmallBitReader(r)
	}
}

// aborted reports whether parsing is being aborted: a fatal error or a handler panic has been recorded
// or the context passed to ParseToEnd() is done, see abortState.
// Message handlers no-op once it is set, so the queued backlog drains without further decoding.
func (p *Parser) aborted() bool {
	return p.abort.aborted()
}

// setError records a fatal error, see abortState.setError().
func (p *Parser) setError(err error) {
	p.abort.setError(err)
}

// handlePanic is the PanicHandler of the message dispatcher, it's called on the message-handling goroutine.
// Handler panics are recorded via setPanic(), anything else is a decoding failure.
// It must never panic, otherwise the message-handling goroutine dies and parsing deadlocks.
func (p *Parser) handlePanic(r any) {
	if classifyPanic(r) == panics.KindUser {
		p.abort.setPanic(panics.Capture(r))

		return
	}

	p.abort.setError(newDecodeError(panics.NewFailure(r), p.currentFrame))
}

func (p *Parser) unregisterAllHandlers() {
	if p.msgDispatcher != nil {
		p.msgDispatcher.UnregisterAllHandlers()
	}

	if p.eventDispatcher != nil {
		p.eventDispatcher.UnregisterAllHandlers()
	}
}

func (p *Parser) poolBitReader(r *bit.BitReader) {
	err := r.Pool()
	if err != nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: err.Error(),
		})
	}
}

// NewParser creates a new Parser.
// The options are applied on top of the default configuration, see the option package.
//
// The demostream io.Reader (e.g. os.File or bytes.Reader) must provide demo data in the '.DEM' format,
// unless it's a *cstv.Reader: then the Parser reads the live CSTV broadcast frame format.
// Only a *cstv.Reader passed directly is detected, not one wrapped in another reader (e.g. bufio.Reader).
// The stream isn't read until parsing starts (see ParseToEnd() and ParseNextFrame()).
// If it implements io.Closer, Parser.Close() closes it.
//
// Example for a live CSTV broadcast:
//
//	r, err := cstv.NewReader(ctx, "http://localhost:8080/s85568392932860274t1733091777")
//	if err != nil {
//		return err
//	}
//
//	p := demoinfocs.NewParser(r)
//	defer p.Close()
//
//	err = p.ParseToEnd(ctx)
//
// See also: ParseFile()
func NewParser(demostream io.Reader, opts ...option.Option) *Parser {
	cfg := config.New(opts)

	if _, ok := demostream.(*cstv.Reader); ok {
		cfg.Format = config.FormatCSTVBroadcast
	}

	return newParserFromConfig(demostream, cfg)
}

// ParserCallback configures a Parser (e.g. registers handlers) before ParseFile() parses the demo.
type ParserCallback func(*Parser) error

// ParseFile parses a demo file at the given path.
// The handler is called with the Parser instance before parsing starts.
// The options are applied on top of the default configuration, see the option package.
//
// Returns an error if the file can't be opened, if the parser encounters an error or ctx is done before the end of the demo.
// Panics with a *PanicError if a handler panics, see Parser.ParseToEnd().
func ParseFile(ctx context.Context, path string, configure ParserCallback, opts ...option.Option) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer f.Close()

	p := NewParser(f, opts...)
	defer p.Close()

	err = configure(p)
	if err != nil {
		return fmt.Errorf("failed to configure parser: %w", err)
	}

	err = p.ParseToEnd(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse demo: %w", err)
	}

	return nil
}

//nolint:funlen
func newParserFromConfig(demostream io.Reader, cfg config.Config) *Parser {
	var p Parser

	// Init parser
	p.config = cfg
	p.demostream = demostream

	p.rawPlayers = make(map[int]*common.PlayerInfo)
	p.dirtyControllers = make(map[int]*dirtyPlayerController)
	p.provisionalNames = make(map[*common.Player]struct{})
	p.triggers = make(map[int]*boundingBoxInformation)
	p.demoInfoProvider = demoInfoProvider{parser: &p}
	p.gameState = newGameState(p.demoInfoProvider)
	p.equipmentTypePerModel = make(map[uint64]common.EquipmentType)
	p.userCmdStates = make(map[int32]*userCmdPlayerState)
	p.userCmdButtonStates = make(map[int32]*userCmdButtonPlayerState)
	p.gameEventHandler = newGameEventHandler(&p, cfg.IgnoreErrBombsiteIndexNotFound)
	p.bombsiteA.index = -1
	p.bombsiteB.index = -1
	p.recordingPlayerSlot = -1
	p.snapshotTick = -1
	p.disableMimicSource1GameEvents = cfg.DisableMimicSource1Events
	p.eagerVelocityTracking = cfg.EagerVelocityTracking
	p.source2FallbackGameEventListBin = cfg.Source2FallbackGameEventListBin
	p.ignorePacketEntitiesPanic = cfg.IgnorePacketEntitiesPanic
	p.decryptionKey = cfg.NetMessageDecryptionKey

	// Only the message dispatcher has a queue, events are dispatched synchronously
	// (their handlers' panics reach the PanicHandler through the net-message handler that dispatched them,
	// or ParseToEnd() / ParseNextFrame() on the parsing goroutine).
	p.msgDispatcher = dp.NewDispatcherWithConfig(dp.Config{PanicHandler: p.handlePanic})
	p.eventDispatcher = new(dp.Dispatcher)
	p.abort.onStop = p.unregisterAllHandlers

	// Created up front so ServerClasses() values obtained before the header is parsed stay live.
	p.stParser = st.NewParser()
	p.stParser.OnEntity(p.onEntity)

	registerInternal(&p, p.handleGameEventList)
	registerInternal(&p, p.handleGameEvent)
	registerInternal(&p, p.handleServerInfo)
	registerInternal(&p, p.handleCreateStringTable)
	registerInternal(&p, p.handleUpdateStringTable)
	registerInternal(&p, p.handleSetConVar)
	registerInternal(&p, p.handleServerRankUpdate)
	registerInternal(&p, p.handleMessageSayText)
	registerInternal(&p, p.handleMessageSayText2)
	registerInternal(&p, p.handleEncryptedData)
	registerInternal(&p, p.handleSendTables)
	registerInternal(&p, p.handleFileInfo)
	registerInternal(&p, p.handleDemoFileHeader)
	registerInternal(&p, p.handleClassInfo)
	registerInternal(&p, p.handleStringTables)
	registerInternal(&p, p.handleFrameParsed)
	registerInternal(&p, p.handleBrokenFullPacket)
	registerInternal(&p, p.gameState.handleIngameTickNumber)
	registerInternal(&p, p.handleUserCommands)

	if cfg.MsgQueueBufferSize >= 0 {
		p.initMsgQueue(cfg.MsgQueueBufferSize)
	}

	p.additionalNetMessageCreators = cfg.AdditionalNetMessageCreators

	return &p
}

func (p *Parser) initMsgQueue(buf int) {
	p.msgQueue = make(chan any, buf)
	p.msgDispatcher.AddQueues(p.msgQueue)
}

type demoInfoProvider struct {
	parser *Parser
}

func (p demoInfoProvider) IngameTick() int {
	return p.parser.gameState.IngameTick()
}

func (p demoInfoProvider) TickRate() float64 {
	return p.parser.TickRate()
}

func (p demoInfoProvider) EagerVelocityTracking() bool {
	return p.parser.eagerVelocityTracking
}

func (p demoInfoProvider) SnapshotTick() int {
	return p.parser.snapshotTick
}

func (p demoInfoProvider) ProcessingSnapshot() bool {
	return p.parser.processingSnapshot
}

func (p demoInfoProvider) TrackFramePosition(record func() bool, lazy bool) {
	p.parser.framePositionRecorders = append(p.parser.framePositionRecorders, record)

	if lazy && !p.parser.lazyVelocityTrackingWarned {
		p.parser.lazyVelocityTrackingWarned = true

		p.parser.eventDispatcher.Dispatch(events.ParserWarn{
			Message: "Player.Velocity() called on a demo without networked player velocity; deriving it from position " +
				"deltas from now on - it's zero for a newly tracked player until the next entity snapshot has been applied. " +
				"Use option.EagerVelocityTracking(true) to record all players from the start.",
			Type: events.WarnTypeVelocityTrackingStartedLazily,
		})
	}
}

// NetworkProtocol returns the demo's network protocol (patch version), 0 if not (yet) known.
func (p demoInfoProvider) NetworkProtocol() int {
	if p.parser.header == nil {
		return 0
	}

	return p.parser.header.NetworkProtocol
}

func (p demoInfoProvider) RecordingPlayerSlot() int {
	return p.parser.recordingPlayerSlot
}

func (p demoInfoProvider) FindPlayerByHandle(handle uint64) *common.Player {
	return p.parser.gameState.Participants().FindByHandle64(handle)
}

func (p demoInfoProvider) FindPlayerByPawnHandle(handle uint64) *common.Player {
	return p.parser.gameState.Participants().FindByPawnHandle(handle)
}

func (p demoInfoProvider) FindEntityByHandle(handle uint64) *st.Entity {
	return p.parser.gameState.EntityByHandle(handle)
}

func (p demoInfoProvider) FindEntityByID(id int) *st.Entity {
	return p.parser.gameState.entities[id]
}

func (p demoInfoProvider) FindWeaponByEntityID(entityID int) *common.Equipment {
	return p.parser.gameState.weapons[entityID]
}
