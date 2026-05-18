# Twitter Clone
Twitter feed back end architecture.
## Tech Stack
* **Language:** Go 1.25 
* **Database:** CockroachDB 
* **Message Broker:** RabbitMQ 
* **Infrastructure:** Docker & Docker Compose
* **Pagination:** Base64 Opaque Cursor Pagination
## Quick Start
This project uses Docker to orchestrate the entire distributed cluster.
### Prerequisites
* Docker installed and running 
* Git installed
### Installation
1. **Clone the repository:**
```bash
git clone https://github.com/metrAJ/go-twitter-clone
cd go-twitter-clone
```
2. **Start:**
* Run the provided boot script:
```bash
./start.sh
```
* or use `make`
```bash
make run
```
### Environment Variables
The system relies on a `.env` file for configuration. You can make you own from `.env.example` or just start the script and it will use the values from `.env.example` to run the project. 
## API Endpoints
The API runs on `http://localhost:3000` by default.
### 1. Post a Message
* URL: `http://localhost:3000/message`
* Method: POST
* Body:
```JSON
{
    "content": "Hello world!"
}
```
### 2. Fetch the Feed
* URL: `http://localhost:3000/feed`
* Method: GET
* Response:
```JSON
{
    "messages": [
        {
            "id": "uuid-here",
            "content": "Hello world!",
            "created_at": "2026-05-18T12:00:00Z"
        }
    ],
    "next_cursor": "QWE..."
}
```
* To fetch the next page: Append the cursor to the query string: `/feed?cursor=QWE...`
### 3. Real-Time Live Feed
* URL: `http://localhost:3000/feed/stream`
* Method: GET
* Response:
<img width="933" height="546" alt="image" src="https://github.com/user-attachments/assets/1ad896b2-4b8c-4335-b735-bcf0f5c27d11" />

## Automated Load Testing
Bot container automatically starts alongside the API. Once the API is healthy, the bot fires 5 (or your specified amount of) messages per second into the system.
## Shutting Down
If you want to remove all the containers and cleare mounts just use: 
```Bash
make dowm
```








