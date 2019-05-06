FROM golang:1.12

COPY dep-*.sha256 .

RUN mkdir release && \
    wget -q -O release/dep-$(go env GOOS)-$(go env GOARCH) https://github.com/golang/dep/releases/download/v0.5.0/dep-$(go env GOOS)-$(go env GOARCH) && \
    sha256sum -c dep-$(go env GOOS)-$(go env GOARCH).sha256 && \
    mv release/dep-$(go env GOOS)-$(go env GOARCH) /go/bin/dep && \
    chmod +x /go/bin/dep && \
    rmdir release && \
    rm dep-*.sha256

ENTRYPOINT ["make"]
