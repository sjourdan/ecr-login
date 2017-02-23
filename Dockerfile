FROM scratch
MAINTAINER Stephane Jourdan <sjourdan@greenalto.com>

COPY certs/ca-certificates.crt /etc/ssl/certs/
COPY ecr-login /

CMD ["/ecr-login"]
