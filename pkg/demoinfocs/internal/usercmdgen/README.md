# usercmdgen

Generates `pkg/demoinfocs/s2_usercmd_typed.go`: a typed deep copy and a typed delta merge for
`msg.CSGOUserCmdPB` and every message it contains. They replace the `proto.Clone` and protoreflect-based
merge on the user-command path, see `applyUserCmdDelta` in `../s2_usercmd_delta.go`.

## Source of truth

The generator reads the Go types of package `msg` (their protobuf struct tags) and their descriptors, so
the typed file is derived from the generated `.pb.go` files, not from the `.proto` files. The two are
regenerated together:

```sh
go generate ./pkg/demoinfocs
```

`../msg/generate.sh` ends with that step, so the nightly "Update Protobufs" workflow regenerates the
typed file in the same change as the protos. `TestGeneratedCodeIsCurrent` fails while the checked-in file
is stale, and `TestGeneratedCodeCoversEveryField` fails if a wire field is dropped.

## Supported subset

The generator handles what these messages need and rejects everything else, so a new field kind stops
generation instead of being miscompiled:

- pointer scalars, `[]byte`, nested messages, and repeated message fields
- proto2 messages only (`proto.Unmarshal` validates a proto3 string's UTF-8, which the merge doesn't)
- oneofs, groups, required fields, zigzag (`sint`), repeated scalars, and types outside package `msg`
  are unsupported and fail with an error naming the field

If an error names a field Valve added to one of these messages, extend the generator and run the command
again. A failed generator also fails the scheduled protobuf update, which opens an issue - the repo falls
behind SteamDatabase until it is extended.

## The contract

`mergeUserCmdDeltaReflect` in `../s2_usercmd_delta_reflect_test.go` is the frozen reference: the
reflective merge this generator replaced. `TestUserCmdTypedMergeMatchesReflect` and
`FuzzApplyUserCmdDelta` compare the typed merge against it on random and malformed deltas. Do not change
the reference to match the generated code - change the generated code.
