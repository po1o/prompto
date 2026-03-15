# See here for image contents: https://github.com/devcontainers/images/blob/main/src/go/.devcontainer/Dockerfile

# [Choice] Go version: 1, 1.26, 1.25, 1-trixie, 1.26-trixie, 1.25-trixie, 1-bookworm, 1.26-bookworm, 1.25-bookworm
ARG VARIANT=2-1.26-trixie
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# Install powershell
ARG PS_VERSION="7.5.4"
ARG PROTOC_VERSION="33.4"
# powershell-7.5.4-linux-x64.tar.gz
# powershell-7.5.4-linux-arm64.tar.gz
RUN ARCH="$(dpkg --print-architecture)"; \
    if [ "${ARCH}" = "amd64" ]; then \
        PS_BIN="v$PS_VERSION/powershell-$PS_VERSION-linux-x64.tar.gz"; \
    elif [ "${ARCH}" = "arm64" ]; then \
        PS_BIN="v$PS_VERSION/powershell-$PS_VERSION-linux-arm64.tar.gz"; \
    elif [ "${ARCH}" = "armhf" ]; then \
        PS_BIN="v$PS_VERSION/powershell-$PS_VERSION-linux-arm32.tar.gz"; \
    fi; \
    wget https://github.com/PowerShell/PowerShell/releases/download/$PS_BIN -O pwsh.tar.gz; \
    mkdir /usr/local/pwsh && \
    tar Cxvfz /usr/local/pwsh pwsh.tar.gz && \
    rm pwsh.tar.gz && \
    chmod +x /usr/local/pwsh/pwsh

RUN echo 'deb http://download.opensuse.org/repositories/shells:/fish:/release:/4/Debian_13/ /' | tee /etc/apt/sources.list.d/shells:fish:release:4.list; \
    curl -fsSL https://download.opensuse.org/repositories/shells:fish:release:4/Debian_13/Release.key | gpg --dearmor | tee /etc/apt/trusted.gpg.d/shells_fish_release_4.gpg > /dev/null; \
    apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
    fish \
    tmux \
    fzf \
    unzip \
    && apt-get clean

ARG USERNAME=vscode

RUN ARCH="$(dpkg --print-architecture)"; \
    case "${ARCH}" in \
      amd64) PROTOC_ARCHIVE="protoc-${PROTOC_VERSION}-linux-x86_64.zip" ;; \
      arm64) PROTOC_ARCHIVE="protoc-${PROTOC_VERSION}-linux-aarch_64.zip" ;; \
      *) echo "unsupported architecture: ${ARCH}" && exit 1 ;; \
    esac; \
    mkdir -p /usr/local/protoc; \
    wget -q "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ARCHIVE}" -O /tmp/protoc.zip; \
    unzip -q /tmp/protoc.zip -d /usr/local/protoc; \
    rm /tmp/protoc.zip

ENV PATH=$PATH:/usr/local/pwsh:/usr/local/protoc/bin:/home/${USERNAME}/bin

RUN GOBIN=/usr/local/protoc/bin go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11; \
    GOBIN=/usr/local/protoc/bin go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1; \
    GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
    GOBIN=/usr/local/bin go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
    GOBIN=/usr/local/bin go install golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest; \
    protoc --version; \
    protoc-gen-go --version; \
    protoc-gen-go-grpc --version; \
    golangci-lint version; \
    fieldalignment -help >/dev/null; \
    modernize -help >/dev/null

# Setup a neat little PowerShell experience
RUN pwsh -Command Install-Module prompto-git -Scope AllUsers -Force; \
    pwsh -Command Install-Module z -Scope AllUsers -Force; \
    pwsh -Command Install-Module PSFzf -Scope AllUsers -Force; \
    pwsh -Command Install-Module Terminal-Icons -Scope AllUsers -Force;

# Deploy prompto prompt to Powershell:
COPY Microsoft.PowerShell_profile.ps1 /home/${USERNAME}/.config/powershell/Microsoft.PowerShell_profile.ps1

# Deploy prompto prompt to Fish:
COPY config.fish /home/${USERNAME}/.config/fish/config.fish

# Everything runs as root during build time, so we want
# to make sure the vscode user can edit these paths too:
RUN chmod 777 -R /home/${USERNAME}/.config

# Override vscode's own Bash prompt with prompto:
RUN sed -i 's/^__bash_prompt$/#&/' /home/${USERNAME}/.bashrc && \
    echo "eval \"\$(prompto init bash)\"" >> /home/${USERNAME}/.bashrc

# Override vscode's own ZSH prompt with prompto:
RUN echo "eval \"\$(prompto init zsh)\"" >> /home/${USERNAME}/.zshrc

# Set container timezone:
ARG TZ="UTC"
RUN ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime

# [Optional] Uncomment the next line to use go get to install anything else you need
# RUN go get -x github.com/JanDeDobbeleer/battery
