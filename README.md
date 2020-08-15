# ghpull

Auto git-pull by GitHub WebHook

## Installation

- Environment in this case
    |Description|Value|
    |-|-|
    |GitHub Repository|`https://github.com/RyuaNerin/portfolio`|
    |GitHub WebHook Payload URL|`https://ryuar.in/__push`|
    |GitHub WebHook Secret|`1234567890`|
    |Local Repository Directory|`/srv/http/ryuar.in/_default`|
    |SSH Key|`/home/ghpull/.ssh/id_ed25519`|
    |HTTP Server Binding|tcp `:8081`|

1. Config Github Actions
    1. **Repository Page** -> **Settings** -> **Webhooks**
    1. Click **Add webhook**
    1. Input `https://ryuar.in/__push` in **Payload URL**
    1. Input `1234567890` in **Secret**
    1. Select **Just the push event.**
    1. Click `Add Webhook.`

1. Clone target repository.

    ```shell
    > git clone https://github.com/RyuaNerin/portfolio -o /srv/http/ryuar.in/_default
    ```

1. Clone this repository, then build package

    ```shell
    > git clone https://github.com/RyuaNerin/ghpull.git
    > cd ghpull
    > go build -v
    ```

1. Copy binary.

    ```shell
    > cp ghpull /usr/local/bin/ghpull
    ```

1. Run ghpull and test.

    ```shell
    > /usr/local/bin/ghpull -key "/home/ghpull/.ssh/id_ed25519" -unix "/run/ghpull/ryuar.in.sock" -path "/__push" -dir "/srv/http/ryuar.in/_default" -secret "1234567890"
    ```

1. To Use systemd.

    ```shell
    > sudo cp ghpull.service /etc/systemd/system/
    > systemctl enable ghpull.service
    > systemctl start ghpull.service
    ```

1. Edit Nginx Configure.

    ```shell
    > vi /etc/nginx/conf.d/ryuar.in.conf
    ```

    ```nginx
    location /__push {
        include fastcgi_params;
        fastcgi_pass unix:/run/ghpull/ryuar.in.sock;
    }
    ```
