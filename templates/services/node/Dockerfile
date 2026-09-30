FROM node:22-alpine
WORKDIR /app
COPY server.js .
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["node", "server.js"]
