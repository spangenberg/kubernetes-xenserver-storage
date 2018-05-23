FROM centos:7
RUN curl -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 --output /usr/local/bin/jq \
 && chmod +x /usr/local/bin/jq
COPY deployer xenserver /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/deployer"]
