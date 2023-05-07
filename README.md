# Overview

First request:
1. Authenticate against Unity
2. Retrieve auth cookies
3. Return auth cookies to client (browser)
4. Client is now authenticated for all subsequent requests to Unity API
5. A DB entry is created for the user containing the publisher id

Note: this will not work for caching -> we need to be able to verify authenticated access to the cache separately from access to the Unity API.
Could create a separate JWT which we can validate for each request. Either the Unity tokens or the JWT might expire earlier, we cannot sync this. 

Next requests:
1. Client sends requests with auth cookies
2. For each API requests, the API service checks the cache first
3. API service forwards requests to Unity API on cache miss and populates the cache

If the Unity API returns a 401, invalidate the JWT somehow (clear the cookie).
Additionally send out a message which the scheduler may react to (stop caching).

Scheduler:
1. When user is first created, a message is sent to the scheduler
2. The scheduler periodically sends a request to the API service to fetch ALL sales data of all months
3. The scheduler then populates the cache with the data

The scheduler receives the JWT from the API service.
When the JWT expires, or any Unity API returns a 401, the scheduler stops fetching data for that particular publisher.

Cache:
1. Stores all sales data per month by publisher id
2. On a request to cache new (different) data, sends a message to the notification service to inform the user of new sales


## To do's

* Add JWT verification to all services
* Move authentication to API gateway
* Convert API gateway to use Gin rather than Fiber
* Return frontend from API gateway
* Run on k8s
