FROM node:20-slim

RUN adduser --home /sandbox --disabled-password sandbox

USER sandbox
WORKDIR /sandbox

ENV NODE_ENV=production
ENV NODE_OPTIONS="--no-warnings"
