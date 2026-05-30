package index

// #cgo CFLAGS: -DSQLITE_CORE -DSQLITE_ENABLE_FTS5 -I${SRCDIR}/cgo_headers
// #cgo LDFLAGS: -lm
// #include "sqlite-vec.c"
import "C"

func initVec() {
	C.sqlite3_auto_extension((*[0]byte)((C.sqlite3_vec_init)))
}
