FROM alpine:latest@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS prep
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
ARG USER_GID=10001
USER ${USER_UID}:${USER_GID}

COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY nrdotcol /
EXPOSE 4317 55680 55679
ENTRYPOINT ["/nrdotcol"]
CMD ["--config", "/etc/otel/config.yaml"]
