.PHONY: build clean

build: ~/.local/bin
	go build -o ~/.local/bin/p ./main.go
	mkdir -p ~/.local/share/bash-completion/completions
	p completion bash > ~/.local/share/bash-completion/completions/p

clean:
	rm -f ~/.local/bin/p
