FROM quay.io/debezium/connect:3.6.0.Beta1

USER root

COPY kafka/connect/apicurio-registry-distro-connect-converter \
     /kafka/connect/apicurio-registry-distro-connect-converter

RUN chown -R kafka:kafka /kafka/connect/apicurio-registry-distro-connect-converter

USER kafka