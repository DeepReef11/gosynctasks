#!/bin/bash
set -e

# Find or build binary
BINARY=""
if [ -f "./gosynctasks" ] && [ -x "./gosynctasks" ]; then
    BINARY="./gosynctasks"
elif [ -f "./gosynctasks/gosynctasks" ] && [ -x "./gosynctasks/gosynctasks" ]; then
    BINARY="./gosynctasks/gosynctasks"
else
    BINARY="$(which gosynctasks 2>/dev/null || echo "")"
fi

if [ -z "$BINARY" ] || [ ! -f "$BINARY" ]; then
    if [ -f "cmd/gosynctasks/main.go" ]; then
        echo "Building gosynctasks..."
        go build -o gosynctasks_bin ./cmd/gosynctasks
        BINARY="./gosynctasks_bin"
    else
        echo "Error: gosynctasks not found and cannot build"
        exit 1
    fi
fi

SHELL_NAME=$(basename "$SHELL")
echo "Detected shell: $SHELL_NAME"

case "$SHELL_NAME" in
    bash)
        COMP_DIR="${HOME}/.bash_completion.d"
        mkdir -p "$COMP_DIR"
        COMP_FILE="$COMP_DIR/gosynctasks"
        $BINARY completion bash > "$COMP_FILE"

        BASHRC="$HOME/.bashrc"
        if [ -f "$BASHRC" ] && ! grep -q "gosynctasks" "$BASHRC" 2>/dev/null; then
            echo -e "\n# gosynctasks completion\n[ -f $COMP_FILE ] && source $COMP_FILE" >> "$BASHRC"
        fi
        echo "✓ Installed to $COMP_FILE"
        echo "Run: source ~/.bashrc"
        ;;

    zsh)
        COMP_DIR="$HOME/.zsh/completion"
        mkdir -p "$COMP_DIR"
        COMP_FILE="$COMP_DIR/_gosynctasks"
        $BINARY completion zsh > "$COMP_FILE"

        ZSHRC="${ZDOTDIR:-$HOME}/.zshrc"
        NEEDS_UPDATE=false
        [ -f "$ZSHRC" ] && ! grep -q "$COMP_DIR" "$ZSHRC" 2>/dev/null && NEEDS_UPDATE=true
        [ -f "$ZSHRC" ] && ! grep -q "compinit" "$ZSHRC" 2>/dev/null && NEEDS_UPDATE=true

        if [ "$NEEDS_UPDATE" = true ]; then
            cat >> "$ZSHRC" << EOF

# gosynctasks completion
fpath=($COMP_DIR \$fpath)
autoload -Uz compinit && compinit
EOF
        fi
        echo "✓ Installed to $COMP_FILE"
        echo "Run: source ~/.zshrc"
        ;;

    fish)
        COMP_DIR="$HOME/.config/fish/completions"
        mkdir -p "$COMP_DIR"
        COMP_FILE="$COMP_DIR/gosynctasks.fish"
        $BINARY completion fish > "$COMP_FILE"
        echo "✓ Installed to $COMP_FILE"
        echo "Restart terminal or run: source ~/.config/fish/config.fish"
        ;;

    *)
        echo "Unsupported shell: $SHELL_NAME (supported: bash, zsh, fish)"
        echo "Manual: $BINARY completion {bash|zsh|fish}"
        exit 1
        ;;
esac
