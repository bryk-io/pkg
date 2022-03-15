# See here for image contents:
# https://github.com/microsoft/vscode-dev-containers/tree/v0.166.0/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version (use -bullseye variants on local arm64/Apple Silicon): 1.16, 1.17, 1.16-bullseye, 1.17-bullseye
ARG VARIANT=1.17
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# [Choice] Node.js version: none, lts/*, 16, 14, 12, 10
ARG NODE_VERSION="none"
RUN if [ "${NODE_VERSION}" != "none" ]; then su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1

# ==== Install additional tools ====

# os packages
RUN \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get -y install \
  icu-devtools

# Live share linux requirements
# https://docs.microsoft.com/en-us/visualstudio/liveshare/use/install-live-share-visual-studio-code#install-linux-prerequisites
RUN \
  wget -O ~/vsls-reqs https://aka.ms/vsls-linux-prereq-script && chmod +x ~/vsls-reqs && ~/vsls-reqs

# protoc and libraries
ARG PROTOBUF_VERSION="3.19.4"
RUN \
  mkdir -p /tmp/protoc && \
  curl -sSL \
    "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-linux-x86_64.zip" \
    -o /tmp/protoc/protoc.zip && \
  cd /tmp/protoc && \
  unzip protoc.zip && \
  mv /tmp/protoc/include /usr/local/include && \
  mv /tmp/protoc/bin/protoc /usr/local/bin/protoc && \
  chmod +x /usr/local/bin/protoc && \
  rm -rf /tmp/protoc

# protoc plugins
ARG PROTOC_GRPC_GATEWAY="2.7.3"
ARG PROTOC_GEN_GO_GRPC="1.2.0"
ARG PROTOC_GEN_VALIDATE="0.6.3"
ARG PROTOC_GEN_GO="1.27.1"
ARG PROTOC_GEN_DRPC="0.0.29"
ARG PROTOC_GEN_ENT="0.2.0"
RUN \
  su vscode -c "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v${PROTOC_GRPC_GATEWAY}" && \
  su vscode -c "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v${PROTOC_GRPC_GATEWAY}" && \
  su vscode -c "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOC_GEN_GO_GRPC}" && \
  su vscode -c "go install github.com/envoyproxy/protoc-gen-validate@v${PROTOC_GEN_VALIDATE}" && \
  su vscode -c "go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO}" && \
  su vscode -c "go install storj.io/drpc/cmd/protoc-gen-go-drpc@v${PROTOC_GEN_DRPC}" && \
  su vscode -c "go install entgo.io/contrib/entproto/cmd/protoc-gen-ent@v${PROTOC_GEN_ENT}"

# buf
ARG BUF_VERSION="1.1.0"
RUN \
  curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-$(uname -s)-$(uname -m).tar.gz" | tar -xvzf - -C "/usr/local" --strip-components 1

# grpcurl
ARG GRPCURL_VERSION="1.8.6"
RUN \
  curl -sSL "https://github.com/fullstorydev/grpcurl/releases/download/v${GRPCURL_VERSION}/grpcurl_${GRPCURL_VERSION}_linux_x86_64.tar.gz" -o grpcurl.tgz && \
  tar -xvzf grpcurl.tgz && \
  chmod +x grpcurl && \
  mv grpcurl /usr/local/bin/. && \
  rm LICENSE grpcurl.tgz

# ghz
ARG GHZ_VERSION="0.106.1"
RUN \
  curl -sSL https://github.com/bojand/ghz/releases/download/v${GHZ_VERSION}/ghz-linux-x86_64.tar.gz -o ghz.tgz && \
  tar -xvzf ghz.tgz && \
  rm LICENSE ghz.tgz && \
  chmod +x ghz* && \
  mv ghz* /usr/local/bin/.

# helm
ARG HELM_VERSION="3.8.0"
RUN \
  # helm
  curl -sSL https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz -o helm.tgz && \
  tar -xvzf helm.tgz && \
  mv linux-amd64/helm /usr/local/bin/. && \
  rm -rf linux-amd64 helm.tgz

# kubectl
ARG KUBECTL_VERSION="1.23.4"
RUN \
  # kubectl
  curl -sSL https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl -o kubectl && \
  chmod +x kubectl && \
  mv kubectl /usr/local/bin/.

# additional tools
ARG NANCY_VERSION="1.0.30"
ARG GQLGEN_VERSION="0.16.0"
RUN \
  # goimports
  su vscode -c "go get golang.org/x/tools/cmd/goimports" && \
  # nancy
  curl -sSL https://github.com/sonatype-nexus-community/nancy/releases/download/v${NANCY_VERSION}/nancy-v${NANCY_VERSION}-linux-amd64 -o /usr/local/bin/nancy && \
  chmod +x /usr/local/bin/nancy && \
  # gqlgen
  su vscode -c "go install github.com/99designs/gqlgen@v${GQLGEN_VERSION}"
