FROM centos:7
RUN yum install -y stunnel
COPY xe-cli-7.1.0-137263c.x86_64.rpm /tmp/xe-cli-7.1.0-137263c.x86_64.rpm
RUN rpm -i /tmp/xe-cli-7.1.0-137263c.x86_64.rpm \
 && rm -f /tmp/xe-cli-7.1.0-137263c.x86_64.rpm
ENTRYPOINT ["/usr/bin/xe"]
