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
    > sudo cp ghpull /usr/local/bin/ghpull
    ```

1. Edit Nginx Configure.

    ```shell
    > sudo vi /etc/nginx/conf.d/ryuar.in.conf
    ```

    ```nginx
    location /__push {
        proxy_pass http://unix:/run/ghpull/ryuar.in.sock;
    }
    ```

1. Run ghpull and test.

    ```shell
    > sudo -u ghpull /usr/local/bin/ghpull -unix "/run/ghpull/ryuar.in.sock" -path "/__push" -dir "/srv/http/ryuar.in/_default" -secret "1234567890"
    ```

1. To Use systemd.

    ```shell
    > sudo cp ghpull.service /etc/systemd/system/
    > sudo vim /etc/systemd/system/ghpull.service
    > sudo systemctl enable ghpull.service
    > sudo systemctl start ghpull.service
    ```
