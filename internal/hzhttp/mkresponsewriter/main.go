package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"text/template"
)

func main() {
	out := flag.String("o", "", "Output file. If not set, write to stdout.")
	flag.Parse()

	cmd := exec.Command("gofmt")
	rawstream, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if *out == "" {
		cmd.Stdout = os.Stdout
	} else {
		fh, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		cmd.Stdout = fh
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = generate(rawstream)
	if err != nil {
		log.Fatal(err)
	}

	err = rawstream.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func generate(w io.Writer) error {
	_, err := w.Write([]byte(header))
	if err != nil {
		return err
	}

	tmpl, err := template.New("template").Parse(templateStr)
	if err != nil {
		return err
	}

	bools := []bool{true, false}
	for _, hijack := range bools {
		for _, flush := range bools {
			for _, closenotify := range bools {
				name := "responseWriterWrap"
				if hijack {
					name += "Hijack"
				}
				if flush {
					name += "Flush"
				}
				if closenotify {
					name += "CloseNotify"
				}

				args := TemplateArgs{
					Name:        name,
					Hijack:      hijack,
					Flush:       flush,
					CloseNotify: closenotify,
				}

				err = tmpl.Execute(w, args)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var header = `
// NOTE: This file was automatically generated by mkresponsewriter, do not edit directly.

package hzhttp

import (
    "bufio"
    "net"
    "net/http"
    "time"
)

var responseWriterConstructors = map[responseWriterType]func(http.ResponseWriter, *responseWriterState) http.ResponseWriter{}

`

type TemplateArgs struct {
	Name                       string
	Hijack, Flush, CloseNotify bool
}

var templateStr = `
type {{.Name}} struct {
    rws *responseWriterState
    inner http.ResponseWriter
}

func (w *{{.Name}}) Header() http.Header {
    return w.inner.Header()
}

func (w *{{.Name}}) Write(p []byte) (int, error) {
    start := time.Now()
    n, err := w.inner.Write(p)
    w.rws.Transfer.Duration += time.Now().Sub(start)
    w.rws.Transfer.Bytes += int64(n)
    return n, err
}

func (w *{{.Name}}) WriteHeader(status int) {
    w.rws.Status = status
    w.inner.WriteHeader(status)
}

var _ http.ResponseWriter = &{{.Name}}{}

{{if .Hijack}}
	func (w *{{.Name}}) Hijack() (net.Conn, *bufio.ReadWriter, error) {
		c, rw, err := w.inner.(http.Hijacker).Hijack()
		if err != nil && w.rws.Status == 0 {
			w.rws.Status = http.StatusSwitchingProtocols
		}
		return c, rw, err
	}

	var _ http.Hijacker = &{{.Name}}{}
{{end}}

{{if .Flush}}
	func (w *{{.Name}}) Flush() {
		w.inner.(http.Flusher).Flush()
	}

	var _ http.Flusher = &{{.Name}}{}
{{end}}

{{if .CloseNotify}}
	func (w *{{.Name}}) CloseNotify() <-chan bool {
		return w.inner.(http.CloseNotifier).CloseNotify()
	}

	var _ http.CloseNotifier = &{{.Name}}{}
{{end}}

func init() {
	typ := responseWriterType{
		Hijacker: {{.Hijack}},
		Flusher: {{.Flush}},
		CloseNotifier: {{.CloseNotify}},
	}
	responseWriterConstructors[typ] = func (w http.ResponseWriter, rws *responseWriterState) http.ResponseWriter {
		return &{{.Name}}{rws, w}
	}
}
`