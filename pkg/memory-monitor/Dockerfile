FROM lfedge/eve-alpine:0f2e0da38e30753c68410727a6cc269e57ff74f2 as memory-monitor-build

ENV BUILD_PKGS gcc musl-dev make linux-headers cmake build-base
ENV PKGS alpine-baselayout curl strace
ENV PKGS_arm64 procps
ENV PKGS_amd64 procps
RUN eve-alpine-deploy.sh

COPY . /root/memory-monitor

WORKDIR /root/memory-monitor

RUN make dist
RUN chmod +x /root/memory-monitor/dist/memory-monitor-handler.sh

FROM scratch
WORKDIR /
# Create base Linux layout, copying from /out (produced by eve-alpine-deploy.sh)
COPY --from=memory-monitor-build /out/ /
# Copy all the memory-monitor artifacts
COPY --from=memory-monitor-build /root/memory-monitor/dist/memory-monitor /sbin/
COPY --from=memory-monitor-build /root/memory-monitor/dist/memory-monitor-handler.sh /sbin/
COPY --from=memory-monitor-build /root/memory-monitor/dist/memory-monitor.conf /etc/

ENTRYPOINT []
# Run the memory-monitor with the -f flag to run in foreground, so that the container does not exit
# immediately after starting, as LinuxKit does not support running daemon processes.
CMD ["/sbin/memory-monitor", "-f"]



