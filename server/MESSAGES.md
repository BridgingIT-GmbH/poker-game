# Messages

## Registration

* client->server: ```REGISTER;name```
* client->server: ```REGISTER_SPECTATOR;name```
* server->client: ```REGISTERED```
* server->client: ```REJECTED```

## Round Start

* server->clients: ```ROUND STARTING;token```
* client->server: ```JOIN;token```
* server->clients: ```ROUND STARTED;round_number;player_names```
* server->client: ```CARDS;two_cards```

## Round Play

* server->client: ```YOUR TURN;token```
* client->server: ```command;token``` //command=FOLD,CHECK,RAISE,CALL
* server->clients: ```command;player```
* server->clients: ```BOARD;cards```

## Round End
* server->clients: ```SCORE;player:cards```