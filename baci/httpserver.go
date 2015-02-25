package main

import (
	"io"
	"net"
	"net/http"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/coreos/rocket/cas"
)

type ACISendHandler struct {
	key string
}

func (h *ACISendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ds, err := cas.NewStore(opts.StoreDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	rs, err := ds.ReadStream(h.key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer rs.Close()

	w.WriteHeader(http.StatusOK)
	io.Copy(w, rs)

}

func startHttpServer(listener net.Listener, key string, c chan error) {
	go func() {
		acish := &ACISendHandler{key}
		http.Handle("/aci", acish)
		err := http.Serve(listener, nil)
		if err != nil {
			c <- err
		}
	}()
}
