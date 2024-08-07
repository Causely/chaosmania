FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH

COPY ./out/chaosmania-${TARGETOS}-${TARGETARCH} /bin/chaosmania
COPY ./plans /plans
COPY ./scenarios /scenarios

# Create a user group 'chaosmania'
RUN addgroup --system chaosmania -gid 3000

# Create a user 'chaosmania' under 'chaosmania'
RUN adduser --system --home /home/chaosmania -uid 2000 --ingroup chaosmania chaosmania

# Chown all the files to the causely user.
RUN chown -R chaosmania:chaosmania /bin/chaosmania

# Switch to 'chaosmania'
USER 2000

ENTRYPOINT [ "/bin/chaosmania" ]