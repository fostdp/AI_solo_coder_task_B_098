FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir requests

COPY beacon_simulator.py .

ENV API_BASE=http://beacon-server:8080/api
ENV INTERVAL=60
ENV WEATHER=auto
ENV VISIBILITY=

CMD python beacon_simulator.py --weather ${WEATHER} --interval ${INTERVAL} ${VISIBILITY:+--visibility ${VISIBILITY}}
