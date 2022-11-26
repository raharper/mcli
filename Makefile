BINS := bin/machine bin/machined

.PHONY: all clean
all: $(BINS)

clean:
	rm -v $(BINS)

bin/machine: cmd/machine/*.go cmd/machine/cmd/*.go
	go build -o $@ cmd/machine/*.go

bin/machined: cmd/machined/*.go cmd/machined/cmd/*.go
	go build -o $@ cmd/machined/*.go
