OS := $(shell uname)
ROCKSDB_MAJOR_VER=5
ifeq ($(OS),Darwin)
ROCKSDB_SO_FILE=librocksdb.$(ROCKSDB_MAJOR_VER).dylib
else ifeq ($(OS),Linux)
ROCKSDB_SO_FILE=librocksdb.so.$(ROCKSDB_MAJOR_VER)
else
$(error OS type $(OS) not supported)
endif

ROCKSDB_INC_PATH ?=
ROCKSDB_LIB_PATH ?=
# in /usr/local/lib?
ifeq ($(ROCKSDB_LIB_PATH),)
ifeq ($(ROCKSDB_INC_PATH),)
ifneq ($(wildcard /usr/local/lib/$(ROCKSDB_SO_FILE)),)
ifneq ($(wildcard /usr/local/include/rocksdb/c.h),)
$(info rocksdb lib found at /usr/local/lib/$(ROCKSDB_SO_FILE))
ROCKSDB_LIB_PATH=/usr/local/lib
endif
endif
endif
endif

ifeq ($(ROCKSDB_LIB_PATH),)
CDEPS_LDFLAGS=-lrocksdb
else
CDEPS_LDFLAGS=-L$(ROCKSDB_LIB_PATH) -lrocksdb
endif
ifneq ($(ROCKSDB_INC_PATH),)
CGO_CXXFLAGS=CGO_CFLAGS="-I$(ROCKSDB_INC_PATH)"
endif
CGO_LDFLAGS=CGO_LDFLAGS="$(CDEPS_LDFLAGS)"
GOCMD=$(CGO_LDFLAGS) $(CGO_CXXFLAGS) go build -v

all: incr

incr:
	$(GOCMD) -o example-incr-server github.com/dencoded/http2raft/examples/incr

clean:
	@rm -f example-incr-server

.PHONY: incr clean