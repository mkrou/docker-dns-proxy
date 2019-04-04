# docker-dns-proxy

#### Tiny proxy for docker containers


## Quick start
1\. Add label **vhost** to a container
```
    docker run --label vhost=myhost.com -d nginx
``` 
2\. Run the proxy
``` 
docker run -d \
 --network bridge \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -p 8080:8080 \
 mkrou/docker-dns-proxy
```
3\. Add localhost:8080 to your browser/os proxy list   
4\. Nginx from the container has been available now on myhost.com