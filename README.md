# url shortener

It's my first bootcamp progect, so it is only made to practice golang. Many things in this project make little sense from production perspective or may be incomplete and made cetain way to meet task requirements only for me to get familiar with some basic concepts.

This project iplements http and grpc servers for url shortening with simple basic authorization and jwt sessions. 

The service may use in-memory or psql storage.


# http handlers
POST: ``/``
shortens given url, only accepts plain text url

GET: ``/{id}``
returns short url, only accepts plain text url

POST: ``api/shorten``
shortenes given url, but only accepts json

POST: ``/api/shorten/batch``
shortens several urls, accepts json

GET: ``/api/user/urls``
returns all short urls uplodaded by curent user

DELETE: ``/api/user/urls``
deletes several urls, accepts json

GET: ``/ping``
checks if db storage is ready; returns db error if not

GET: ``/api/internal/stats``
returns number of urls shortened
