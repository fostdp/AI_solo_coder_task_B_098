FROM postgis/postgis:16-3.4

ENV POSTGRES_DB=beacon_system
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres

EXPOSE 5432

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 \
    CMD pg_isready -U postgres -d beacon_system
