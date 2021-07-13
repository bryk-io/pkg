# See here for image contents:
# https://github.com/microsoft/vscode-dev-containers/tree/v0.166.0/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version: 1.16, 1.15
ARG VARIANT="1.16"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="false"
ARG NODE_VERSION="lts/*"
RUN \
  if [ "${INSTALL_NODE}" = "true" ]; then \
    su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; \
  fi

# [Optional] Uncomment this section to install additional OS packages.
#RUN \
#  export DEBIAN_FRONTEND=noninteractive && \
#  apt-get update && \
#  apt-get -y install --no-install-recommends \
#  <your-package-list-here>

# [Optional] Uncomment this line to install global node packages.
#RUN \
# su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1

# ==== Install additional tools ====

# protoc and libraries
ARG PROTOBUF_VERSION="3.15.6"
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

# buf
ARG BUF_VERSION="0.40.0"
RUN \
  curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-$(uname -s)-$(uname -m).tar.gz" | tar -xvzf - -C "/usr/local" --strip-components 1

# grpcurl
ARG GRPCURL_VERSION="1.8.0"
RUN \
  curl -sSL "https://github.com/fullstorydev/grpcurl/releases/download/v${GRPCURL_VERSION}/grpcurl_${GRPCURL_VERSION}_linux_x86_64.tar.gz" -o grpcurl.tgz && \
  tar -xvzf grpcurl.tgz && \
  chmod +x grpcurl && \
  mv grpcurl /usr/local/bin/. && \
  rm LICENSE grpcurl.tgz

# ghz
ARG GHZ_VERSION="0.92.0"
RUN \
  curl -sSL https://github.com/bojand/ghz/releases/download/v${GHZ_VERSION}/ghz-linux-x86_64.tar.gz -o ghz.tgz && \
  tar -xvzf ghz.tgz && \
  rm LICENSE ghz.tgz && \
  chmod +x ghz* && \
  mv ghz* /usr/local/bin/.

# protoc-gen-grpc-gateway
ARG GRPC_GATEWAY_VERSION="1.16.0"
RUN \
  curl -sSL \
  https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v${GRPC_GATEWAY_VERSION}/protoc-gen-grpc-gateway-v${GRPC_GATEWAY_VERSION}-linux-x86_64 \
  -o ${GOPATH}/bin/protoc-gen-grpc-gateway && \
  curl -sSL \
  https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v${GRPC_GATEWAY_VERSION}/protoc-gen-swagger-v${GRPC_GATEWAY_VERSION}-linux-x86_64 \
  -o ${GOPATH}/bin/protoc-gen-swagger && \
  chmod +x ${GOPATH}/bin/protoc-gen-grpc-gateway && \
  chmod +x ${GOPATH}/bin/protoc-gen-swagger
  # su vscode -c "go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION}" && \
  # su vscode -c "go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v${GRPC_GATEWAY_VERSION}"

# protoc plugins and additional tools
ARG PROTOC_GEN_GO_GRPC="1.1.0"
ARG PROTOC_GEN_VALIDATE="0.5.0"
ARG PROTOC_GEN_GO="1.26.0"
ARG HELM_VERSION="3.5.3"
ARG KUBECTL_VERSION="1.20.5"
RUN \
  su vscode -c "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOC_GEN_GO_GRPC}" && \
  su vscode -c "go install github.com/envoyproxy/protoc-gen-validate@v${PROTOC_GEN_VALIDATE}" && \
  su vscode -c "go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO}" && \
  # goimports
  su vscode -c "go get golang.org/x/tools/cmd/goimports" && \
  # helm
  curl -sSL https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz -o helm.tgz && \
  tar -xvzf helm.tgz && \
  mv linux-amd64/helm /usr/local/bin/. && \
  rm -rf linux-amd64 helm.tgz && \
  # kubectl
  curl -sLO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
  chmod +x kubectl && \
  mv kubectl /usr/local/bin/.

