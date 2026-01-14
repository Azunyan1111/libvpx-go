//go:build linux && amd64

package vpx

/*
#cgo CFLAGS: -I${SRCDIR}/../include
#cgo LDFLAGS: -L${SRCDIR}/../lib/linux_amd64 -lvpx -lm -lpthread
*/
import "C"
