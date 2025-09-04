

build: ~/.local/bin
	go build -o ~/.local/bin/p .
	mkdir -p ~/.local/share/bash-completion/completions
	p completion bash > ~/.local/share/bash-completion/completions/p
