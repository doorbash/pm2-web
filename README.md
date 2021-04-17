# pm2-web
A simple web based monitor for PM2

<img src="https://github.com/doorbash/pm2-web/blob/master/screenshot.png?raw=true" />

## Build
```
    go build
```

## Usage
```
./pm2-web localhost:3030
```

Set HTTP authentication username password:
```
./pm2-web -u admin -p 1234 localhost:3030
```

Set log buffer size:
```
./pm2-web -l 200 localhost:3030
```

Set process-list update interval (seconds):
```
./pm2-web -i 10 localhost:3030
```

Run using PM2:
```
pm2 start --name pm2-web ./pm2-web -- localhost:3030
```

## Deploying on Nginx:

### Nginx configuration

```
server {
    listen 80;
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate /path/to/your/cert.crt;
    ssl_certificate_key /path/to/your/cert.key;

    location /pm2/logs {
        proxy_pass  http://127.0.0.1:3030/logs;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }

    location /pm2 {
        rewrite ^/pm2$ / break;
        rewrite ^/pm2/(.*)$ /$1 break;    
        proxy_pass  http://127.0.0.1:3030;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## Licecnse
MIT
