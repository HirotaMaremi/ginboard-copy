FROM golang:1.20.3-alpine3.17

COPY ./app /go/src/app

WORKDIR /go/src/app

RUN apk update \
&& apk add tzdata \
&& apk add --no-cache git \
&& go get github.com/gin-gonic/gin \
&& go get github.com/jinzhu/gorm \
&& go get github.com/go-sql-driver/mysql

ENV TZ Asia/Tokyo

EXPOSE 8080
