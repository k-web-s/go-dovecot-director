FROM scratch

ARG TARGETARCH

COPY director.${TARGETARCH} /director

USER 65534

CMD ["/director"]
