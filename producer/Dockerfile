FROM python:3.10

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy only the requirements file and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application code
COPY producer.py .
COPY templates/ templates/

# Expose ports for the Flask app and Prometheus metrics
EXPOSE 5001
EXPOSE 8090

# Create a non-root user and switch to it
RUN groupadd -r appgroup && useradd -r -g appgroup appuser
USER appuser

# Run the script when the container launches
CMD ["python", "./producer.py"]
