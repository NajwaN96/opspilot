FROM node:22-alpine
WORKDIR /app
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build
ENV PORT=3461
ENV HOSTNAME=0.0.0.0
EXPOSE 3461
CMD ["npx", "next", "start", "-p", "3461", "-H", "0.0.0.0"]
