FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH

COPY ./out/chaosmania-${TARGETOS}-${TARGETARCH} /bin/chaosmania
COPY ./plans /plans

# Create a user group 'chaosmania'
RUN addgroup --system chaosmania -gid 3000

# Create a user 'chaosmania' under 'chaosmania'
RUN adduser --system --home /home/chaosmania -uid 2000 --ingroup chaosmania chaosmania

# Chown all the files to the causely user.
RUN chown -R chaosmania:chaosmania /bin/chaosmania

# Switch to 'chaosmania'
USER 2000

ENTRYPOINT [ "/bin/chaosmania" ]