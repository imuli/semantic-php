yml=$(patsubst %.php,%.yml,$(wildcard *.php))
yml5=$(patsubst %.php,%.yml5,$(wildcard *.php))

all: $(yml) $(yml5)

%.yml: %.php
	../semantic-php $^ > $@
%.yml5: %.php
	../semantic-php -php=5 $^ > $@
