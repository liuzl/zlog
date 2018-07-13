package zlog

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/liuzl/filestore"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/rs/zerolog"
)

var (
	dir = flag.String("zlog_dir",
		filepath.Join(filepath.Dir(os.Args[0]), "zerolog"), "zerolog dir")
)

var c Chain
var once sync.Once

func initZlog() {
	once.Do(func() {
		hostname, _ := os.Hostname()
		var out io.Writer
		f, err := filestore.NewFileStore(*dir)
		if err != nil {
			out = os.Stdout
			fmt.Fprintf(os.Stderr, "err: %+v, will zerolog to stdout\n", err)
		} else {
			out = f
		}
		log := zerolog.New(out).With().
			Timestamp().
			Str("service", filepath.Base(os.Args[0])).
			Str("host", hostname).
			Logger()

		c = NewChain()

		// Install the logger handler with default output on the console
		c = c.Append(NewHandler(log))

		c = c.Append(AccessHandler(func(r *http.Request,
			status, size int, duration time.Duration) {
			FromRequest(r).Debug().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Msg("")
		}))

		// Install some provided extra handler to set some request's context fields.
		// Thanks to those handler, all our logs will come with some pre-populated fields.
		c = c.Append(RemoteAddrHandler("ip"))
		c = c.Append(HeaderHandler("X-Forwarded-For"))
		c = c.Append(HeaderHandler("User-Agent"))
		c = c.Append(HeaderHandler("Referer"))
		c = c.Append(RequestIDHandler("req_id", "Request-Id"))
		// keep in order
		c = c.Append(DelResponseHeaderHandler("Cost"))
		c = c.Append(ResponseHeaderHandler("Cost", "float"))
		c = c.Append(DumpResponseHandler("response"))
		c = c.Append(DumpRequestHandler("request"))
	})
}

func WithLog(h httpserver.Handler) httpserver.Handler {
	initZlog()
	return c.Then(h)
}
