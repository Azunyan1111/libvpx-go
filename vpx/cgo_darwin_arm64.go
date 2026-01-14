//go:build darwin && arm64

package vpx

/*
#cgo CFLAGS: -I${SRCDIR}/../include
#cgo LDFLAGS: -L${SRCDIR}/../lib/darwin_arm64 -lvpx
*/
import "C"
