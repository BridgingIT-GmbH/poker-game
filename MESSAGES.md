# Messages

## Registration

* client->server: ```REGISTER;name```
* spectator->server: ```REGISTER_SPECTATOR;name```
* server->client: ```REGISTERED```
* server->client: ```REJECTED```

## Round Start

* server->clients: ```ROUND STARTING;token```
* client->server: ```JOIN;token```
* server->clients: ```ROUND STARTED;round_number;player_names```
* server->client: ```CARDS;two_cards```

## Round Play

* server->client: ```YOUR TURN;token```
* server->spectators: ```TURN;player```
* client->server: ```command;token``` //command=FOLD,CHECK,RAISE,CALL
* server->clients: ```command;player```
* server->clients: ```FOLD;player;reason``` //auto-fold by the server, see below
* server->clients: ```BOARD;cards```
* server->spectators: ```POT;amount```

### Auto-fold reasons

If the server folds a player automatically, the `FOLD` message carries a third
field with a machine-readable reason. Voluntary folds have no reason field.

| Reason               | Meaning                                                  |
|----------------------|----------------------------------------------------------|
| `TIMEOUT`            | Player did not act within the action timeout             |
| `CANNOT_CHECK`       | CHECK sent although there is an outstanding bet          |
| `CANNOT_AFFORD_CALL` | CALL sent without enough chips (all-in not supported)    |
| `INSUFFICIENT_CHIPS` | RAISE sent without enough chips                          |
| `RAISE_LIMIT`        | RAISE sent although the per-street raise limit is reached |
| `INVALID_ACTION`     | Unknown or unexpected action                             |

## Round End
* server->clients: ```SCORE;player1:score:cards:hand_value;player2:score:cards:hand_value;...```
  * Hand values: "HIGH_CARD", "ONE_PAIR", "TWO_PAIR", "THREE_OF_A_KIND", "STRAIGHT", "FLUSH", "FULL_HOUSE", "FOUR_OF_A_KIND", "STRAIGHT_FLUSH"
  * The fields "cards" and "hand_value" are optional: it will not be shown for players who have folded. It will also not be shown for the winning player if the game ends before showdown.
