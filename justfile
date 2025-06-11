PKG_NAME := `basename $(go list)`
BUILD_FLAGS := "-ldflags=\"-s -w\" -trimpath"

PREFIX := env("PREFIX", "/usr/local")

default: install

build:
    go build {{BUILD_FLAGS}} -o {{PKG_NAME}} .

install: build
    mkdir -p {{PREFIX}}/bin/
    mv {{PKG_NAME}} {{PREFIX}}/bin/

uninstall:
    rm -f {{PREFIX}}/bin/{{PKG_NAME}}

clean:
    rm -f {{PKG_NAME}}
