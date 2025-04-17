//
// This is a reference implementation of a playcast webserver.
// It is not suitable for use in production without additional work to
// tailor the implementation to your production environment, to enforce
// proper security around match data uploads, CDN integration, proper
// CDN configuration for HTTP status codes caching rules, proper CDN
// integration with your origin webserver, and numerous other work items.
//
// This implementation is intended to provide a reference vanilla environment
// that other developers can use either as a starting point for their development,
// or to experiment with small incremental changes primarily to help recreate and
// report any issues or feature requests around the playcast systems.
// Having easy steps to reproduce any issues in an environment close to vanilla game
// servers, vanilla game clients, and vanilla reference playcast webserver, is the
// best way to help Valve diagnose and address those issues or get feature requests
// implemented.
//

var http = require( 'http' );
var zlib = require( 'zlib' );
var url = require( 'url' );

"use strict";
var port = 8080;

// In-memory storage of all match broadcast fragments, metadata, etc.
// Can easily hold many matches in-memory on modern computers, especially with compression
var match_broadcasts = {};

// Example of how to support token_redirect (for CDN, unified playcast URL for the whole event, etc.)
var token_redirect_for_example = null;

var stats = {	// Various stats that developers might want to track in their production environments
    post_field: 0, get_field: 0, get_start: 0, get_frag_meta: 0,
    sync: 0, not_found: 0, new_match_broadcasts: 0,
    err: [ 0, 0, 0, 0 ],
    requests: 0, started: Date.now(), version: 1
};


function respondSimpleError( uri, response, code, explanation )
{
    // if( uri ) console.log( uri + " => " + code + " " + explanation );
    response.writeHead( code, { 'X-Reason': explanation } );
    response.end();
}

function checkFragmentCdnDelayElapsed( fragmentRec )
{
    // Validate that any injected CDN delay has elapsed
    if ( fragmentRec.cdndelay )
    {
        if ( !fragmentRec.timestamp )
        {
            console.log( "Refusing to serve cdndelay " + field + " without timestamp" );
            return false;
        }
        else
        {
            var iusElapsedLiveMilliseconds = Date.now().valueOf() - ( fragmentRec.cdndelay + fragmentRec.timestamp.valueOf() );
            if ( iusElapsedLiveMilliseconds < 0 )
            {
                console.log( "Refusing to serve cdndelay " + field + " due to " + iusElapsedLiveMilliseconds + " ms of delay remaining" );
                return false; // refuse to serve the blob due to artificial CDN delay
            }
        }
    }
    return true;
}

function isSyncReady( f )
{
    return f != null && typeof ( f ) == "object" && f.full != null && f.delta != null && f.tick != null && f.endtick != null
        && f.timestamp && checkFragmentCdnDelayElapsed( f );
}

function getMatchBroadcastEndTick( broadcasted_match )
{
    for ( var f = broadcasted_match.length - 1; f >= 0; f-- )
    {
        if ( broadcasted_match[ f ].endtick )
            return broadcasted_match[ f ].endtick;
    }
    return 0;
}

function respondMatchBroadcastSync( param, response, broadcasted_match, token_redirect )
{
    var nowMs = Date.now();
    response.setHeader( 'Cache-Control', 'public, max-age=3' );
    response.setHeader( 'Expires', new Date( nowMs + 3000 ).toUTCString() ); // whatever we find out, this information is going to be stale 3-5 seconds from now
    // TODO: if you use this reference script in production (which you should not), make sure you set all the necessary headers for your CDN to relay the expiration headers to PoPs and clients

    var match_field_0 = broadcasted_match[ 0 ];
    if ( match_field_0 != null && match_field_0.start != null )
    {
        var fragment = param.query.fragment, frag = null;

        if ( fragment == null )
        {
            // skip the last 3-4 fragments, to let the front-running clients get 404, and CDN wait for 3+ seconds, and re-try that fragment again
            // then go back another 3 fragments that are the buffer size for the client - we want to have the full 3 fragments ahead of whatever the user is streaming for the smooth experience
            // if we don't, then legit in-sync clients will often hit CDN-cached 404 on buffered fragments
            fragment = Math.max( 0, broadcasted_match.length - 8 );

            if ( fragment >= 0 && fragment >= match_field_0.signup_fragment )
            {
                // can't serve anything before the start fragment
                var f = broadcasted_match[ fragment ];
                if ( isSyncReady( f ) )
                    frag = f;
            }
        }
        else
        {
            if ( fragment < match_field_0.signup_fragment )
                fragment = match_field_0.signup_fragment;

            for ( ; fragment < broadcasted_match.length; fragment++ )
            {
                var f = broadcasted_match[ fragment ];
                if ( isSyncReady( f ) )
                {
                    frag = f;
                    break;
                }
            }
        }

        if ( frag )
        {
            console.log( "Sync fragment " + fragment );
            // found the fragment that we want to send out
            response.writeHead( 200, { "Content-Type": "application/json" } );
            if ( match_field_0.protocol == null )
                match_field_0.protocol = 5; // Source2 protocol: 5

            var jso = {
                tick: frag.tick,
                endtick: frag.endtick,
                maxtick: getMatchBroadcastEndTick( broadcasted_match ),
                rtdelay: ( nowMs - frag.timestamp ) / 1000, // delay of this fragment from real-time, in seconds
                rcvage: ( nowMs - broadcasted_match[ broadcasted_match.length - 1 ].timestamp ) / 1000, // Receive age: how many seconds since relay last received data from game server
                fragment: fragment,
                signup_fragment: match_field_0.signup_fragment,
                tps: match_field_0.tps,
                keyframe_interval: match_field_0.keyframe_interval,
                map: match_field_0.map,
                protocol: match_field_0.protocol
            };

            if ( token_redirect )
                jso.token_redirect = token_redirect;

            response.end( JSON.stringify( jso ) );
            return; // success!
        }

        // not found
        response.writeHead( 405, "Fragment not found, please check back soon" );
    }
    else
    {
        response.writeHead( 404, "Broadcast has not started yet" );
    }

    response.end();
}

function postField( request, param, response, broadcasted_match, fragment, field )
{
    // decide on what exactly the response code is - we have enough info now
    if ( field == "start" )
    {
        console.log( "Start tick " + param.query.tick + " in fragment " + fragment );
        response.writeHead( 200 );

        if ( broadcasted_match[ 0 ] == null )
            broadcasted_match[ 0 ] = {};
        if ( broadcasted_match[ 0 ].signup_fragment > fragment )
            console.log( "UNEXPECTED new start fragment " + fragment + " after " + broadcasted_match[ 0 ].signup_fragment );

        broadcasted_match[ 0 ].signup_fragment = fragment;
        fragment = 0; // keep the start in the fragment 0
    }
    else
    {
        if ( broadcasted_match[ 0 ] == null )
        {
            console.log( "205 - need start fragment" );
            response.writeHead( 205 );
        }
        else
        {
            if ( broadcasted_match[ 0 ].start == null )
            {
                console.log( "205 - need start data" );
                response.writeHead( 205 );
            }
            else
            {
                response.writeHead( 200 );
            }
        }
        if ( broadcasted_match[ fragment ] == null )
        {
            //console.log("Creating fragment " + fragment + " in match_broadcast " + path[1]);
            broadcasted_match[ fragment ] = {};
        }
    }

    for ( q in param.query )
    {
        var v = param.query[ q ], n = parseInt( v );
        broadcasted_match[ fragment ][ q ] = ( v == n ? n : v );
    }

    var body = [];
    request.on( 'data', function ( data ) { body.push( data ); } );
    request.on( 'end', function ()
    {
        var totalBuffer = Buffer.concat( body );
        if ( field == "start" )
            console.log( "Received [" + fragment + "]." + field + ", " + totalBuffer.length + " bytes in " + body.length + " pieces" );
        response.end(); // we can end the response before gzipping the received data

        var originCdnDelay = request.headers[ 'x-origin-delay' ];
        if ( originCdnDelay && parseInt( originCdnDelay ) > 0 )
        {	// CDN delay must match for both fragments, overwrite is ok
            broadcasted_match[ fragment ].cdndelay = parseInt( originCdnDelay );
        }

        zlib.gzip( totalBuffer, function ( error, compressedBlob )
        {
            if ( error )
            {
                console.log( "Cannot gzip " + totalBuffer.length + " bytes: " + error );
                broadcasted_match[ fragment ][ field ] = totalBuffer;
            }
            else
            {
                //console.log(fragment + "/" + field + " " + totalBuffer.length + " bytes, compressed " + compressedBlob.length + " to " + ( 100 * compressedBlob.length / totalBuffer.length ).toFixed(1) + "%" );
                broadcasted_match[ fragment ][ field + "_ungzlen" ] = totalBuffer.length;
                broadcasted_match[ fragment ][ field ] = compressedBlob;
            }

            // flag the fragment as received and ready for ingestion by CDN (provided "originCdnDelay" is satisfied)
            broadcasted_match[ fragment ].timestamp = Date.now();
        } );
    } );
}

function serveBlob( request, response, fragmentRec, field )
{
    var blob = fragmentRec[ field ];
    var ungzipped_length = fragmentRec[ field + "_ungzlen" ];

    // Validate that any injected CDN delay has elapsed
    if ( !checkFragmentCdnDelayElapsed( fragmentRec ) )
    {
        blob = null; // refuse to serve the blob due to artificial CDN delay
    }

    if ( blob == null )
    {
        response.writeHead( 404, "Field not found" );
        response.end();
    }
    else
    {
        // we have data to serve
        if ( Buffer.isBuffer( blob ) )
        {
            // https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.11
            headers = { 'Content-Type': 'application/octet-stream' };
            if ( ungzipped_length )
            {
                headers[ 'Content-Encoding' ] = 'gzip';
            }
            response.writeHead( 200, headers );
            response.end( blob );
        }
        else
        {
            response.writeHead( 404, "Unexpected field type " + typeof ( blob ) ); // we only serve strings
            console.log( "Unexpected Field type " + typeof ( blob ) ); // we only serve strings
            response.end();
        }
    }
}

function getStart( request, response, broadcasted_match, fragment, field )
{
    if ( broadcasted_match[ 0 ] == null || broadcasted_match[ 0 ].signup_fragment != fragment )
    {
        respondSimpleError( request.url, response, 404, "Invalid or expired start fragment, please re-sync" );
    }
    else
    {
        // always take start data from the 0th fragment
        serveBlob( request, response, broadcasted_match[ 0 ], field );
    }
}

function getField( request, response, broadcasted_match, fragment, field )
{
    serveBlob( request, response, broadcasted_match[ fragment ], field );
}

function getFragmentMetadata( response, broadcasted_match, fragment )
{
    var res = {};
    for ( var field in broadcasted_match[ fragment ] )
    {
        var f = broadcasted_match[ fragment ][ field ];
        if ( typeof ( f ) == 'number' ) res[ field ] = f;
        else if ( Buffer.isBuffer( f ) ) res[ field ] = f.length;
    }
    response.writeHead( 200, { "Content-Type": "application/json" } );
    response.end( JSON.stringify( res ) );
}

function processRequestUnprotected( request, response )
{
    // https://nodejs.org/api/http.html#http_class_http_incomingmessage
    var uri = decodeURI( request.url );

    var param = url.parse( uri, true );
    var path = param.pathname.split( "/" );
    path.shift(); // the first element is always empty, because the path starts with /
    response.httpVersion = '1.0';

    var prime = path.shift();

    if ( prime == null || prime == '' || prime == 'index.html' )
    {
        respondSimpleError( uri, response, 401, 'Unauthorized' );
        return;
    }

    var isPost;
    if ( request.method == 'POST' )
    {
        isPost = true;
        // TODO: if you use this reference script in production (which you should not), make sure you check "originAuth" header - it must match your game server private setting!
        // if ( !verify request.headers['x-origin-auth'] equals "SuPeRsEcUrEsErVeR" ) {
        // 	console.log("Unauthorized POST to " + request.url + ", origin auth " + originAuth);
        // 	respondSimpleError(uri, response, 403, "Not Authorized");
        // 	return;
        // }
    }
    else if ( request.method == 'GET' )
    {
        isPost = false;
        // TODO: if you use this reference script in production (which you should not), make sure you check "originAuth" header - it must match your CDN authorization setting!
        // if ( !verify request.headers['x-origin-auth'] equals "SuPeRsEcUrE_CDN_AuTh" ) {
        // 	respondSimpleError(uri, response, 403, "Not Authorized");
        // 	return;
        // }
    }
    else
    {
        respondSimpleError( uri, response, 404, "Only POST or GET in this API" );
        return;
    }

    var broadcasted_match = match_broadcasts[ prime ];
    if ( broadcasted_match == null )
    {
        // the match_broadcast does not exist
        if ( isPost )
        {
            // TODO: if you use this reference script in production (which you should not), make sure that your intent is to create a new match_broadcast on any POST request
            console.log( "Creating match_broadcast '" + prime + "'" );
            token_redirect_for_example = prime; // TODO: implement your own logic here or somewhere else that decides which token_redirect to use for unified playcast URL/CDN/etc.
            match_broadcasts[ prime ] = broadcasted_match = [];
            stats.new_match_broadcasts++;
        }
        else
        {
            if ( prime == 'sync' )
            {
                // TODO: implement your own logic here or somewhere else that decides which token_redirect to use for unified playcast URL/CDN/etc.
                // This reference implementation (which you should not use in production) will try to redirect to whatever "token_redirect_for_example"
                if ( token_redirect_for_example && match_broadcasts[ token_redirect_for_example ] )
                {
                    respondMatchBroadcastSync( param, response, match_broadcasts[ token_redirect_for_example ], token_redirect_for_example );
                    stats.sync++;
                }
                else
                {
                    respondSimpleError( uri, response, 404, "match_broadcast " + prime + " not found and no valid token_redirect" );
                    stats.err[ 0 ]++;
                }
            }
            else
            {
                // GET requests cannot create new match_broadcasts in this reference implementation
                respondSimpleError( uri, response, 404, "match_broadcast " + prime + " not found" ); // invalid match_broadcast
                stats.err[ 0 ]++;
            }
            return;
        }
    }

    var requestFragmentOrKey = path.shift();
    if ( requestFragmentOrKey == null || requestFragmentOrKey == '' )
    {
        if ( isPost )
        {
            respondSimpleError( uri, response, 405, "Invalid POST: no fragment or field" );
            stats.err[ 1 ]++;
        }
        else
        {
            respondSimpleError( uri, response, 401, "Unauthorized" );
        }
        return;
    }

    stats.requests++;

    var fragment = parseInt( requestFragmentOrKey );

    if ( fragment != requestFragmentOrKey )
    {
        if ( requestFragmentOrKey == "sync" )
        {
            //setTimeout(() => {
            respondMatchBroadcastSync( param, response, broadcasted_match );
            //}, 2000); // can be useful for your debugging additional latency on the /sync response
            stats.sync++;
        }
        else
        {
            respondSimpleError( uri, response, 405, "Fragment is not an int or sync" );
            stats.err[ 2 ]++;
        }
        return;
    }

    var field = path.shift();
    if ( isPost )
    {
        stats.post_field++;
        if ( field != null )
        {
            postField( request, param, response, broadcasted_match, fragment, field );
        }
        else
        {
            respondSimpleError( uri, response, 405, "Cannot post fragment without field name" );
            stats.err[ 3 ]++;
        }
    }
    else
    {
        if ( field == 'start' )
        {
            getStart( request, response, broadcasted_match, fragment, field );
            stats.get_start++;
        }
        else if ( broadcasted_match[ fragment ] == null )
        {
            stats.err[ 4 ]++;
            response.writeHead( 404, "Fragment " + fragment + " not found" );
            response.end();
        }
        else if ( field == null || field == '' )
        {
            getFragmentMetadata( response, broadcasted_match, fragment );
            stats.get_frag_meta++;
        }
        else
        {
            getField( request, response, broadcasted_match, fragment, field );
            stats.get_field++;
        }
    }
}

function processRequest( request, response )
{
    try
    {
        processRequestUnprotected( request, response );
    }
    catch ( err )
    {
        console.log( ( new Date ).toUTCString() + " Exception when processing request " + request.url );
        console.log( err );
        console.log( err.stack );
    }
}

var newServer = http.createServer( processRequest ).listen( port );
if ( newServer )
    console.log( ( new Date() ).toUTCString() + " Started in " + __dirname + " on port " + port );
else
    console.log( ( new Date() ).toUTCString() + " Failed to start on port " + port );
