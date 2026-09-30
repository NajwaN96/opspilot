FROM python:3.12-alpine
WORKDIR /app
COPY app.py .
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["python", "app.py"]
