yml=$(patsubst %.php,%.yml,$(wildcard *.php))

all: $(yml)

%.yml: %.php
	../semantic-php $^ > $@
