FROM alpine:latest
RUN apk add --update nodejs npm
RUN npm i pm2 -g
ADD pm2-web /pm2-web
ADD static /static
EXPOSE 3030
CMD ["pm2-runtime", "--output", "/dev/stdout", "--error", "/dev/stderr", "./pm2-web", "--", "--time", "--app-name", "--actions", ":3030"]