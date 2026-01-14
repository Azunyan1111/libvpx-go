all:
	c-for-go -ccdefs vpx.yml

clean:
	rm -f vpx/cgo_helpers.go vpx/cgo_helpers.h vpx/cgo_helpers.c
	rm -f vpx/const.go vpx/doc.go vpx/types.go
	rm -f vpx/vpx.go

test:
	cd vpx && go test -v

test-linux:
	docker build --platform linux/amd64 -t libvpx-test-linux-amd64 -f Dockerfile.test-linux-amd64 .
	docker run --platform linux/amd64 --rm libvpx-test-linux-amd64

build-libvpx-linux:
	docker build --platform linux/amd64 -t libvpx-builder-linux-amd64 -f Dockerfile.build-linux-amd64 .
	docker run --rm libvpx-builder-linux-amd64 > lib/linux_amd64/libvpx.a
	