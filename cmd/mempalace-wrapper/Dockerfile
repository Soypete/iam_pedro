FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir mempalace

COPY mempalace-wrapper /usr/local/bin/mempalace-wrapper

ENV MEMPALACE_WRAPPER_ADDR=:8082
ENV MEMPALACE_DATA_DIR=/data/palaces

EXPOSE 8082

ENTRYPOINT ["/usr/local/bin/mempalace-wrapper"]