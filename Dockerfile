# Image de plakar-ui : le binaire du service et plakar, dans une version
# figée. Pas de rclone.
#
# Une montée de version de plakar se fait ici, et seulement ici : le test de
# contrat (étage `contract`) la valide à chaque construction.

ARG GO_IMAGE=golang:1.27-alpine
ARG PLAKAR_VERSION=v1.1.6

# plakar, compilé depuis ses sources à la version figée (binaire statique,
# comme les versions publiées par Plakar).
FROM ${GO_IMAGE} AS plakar
ARG PLAKAR_VERSION
RUN CGO_ENABLED=0 GOBIN=/out go install -trimpath github.com/PlakarKorp/plakar@${PLAKAR_VERSION}

# Le service.
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/plakar-ui ./cmd/plakar-ui

# Test de contrat avec le plakar de l'image : la construction échoue si plakar
# a changé le format de ses sorties. PLAKAR_UI_CONTRACT_PLAKAR rend le test
# obligatoire, il ne peut pas être sauté.
FROM build AS contract
COPY --from=plakar /out/plakar /usr/local/bin/plakar
ENV PLAKAR_UI_CONTRACT_PLAKAR=/usr/local/bin/plakar
RUN go test -count=1 -run Contract -v ./internal/app/ \
 && cp /out/plakar-ui /out/plakar-ui.contrat

FROM alpine:3.24
# Certificats pour joindre les Stores en HTTPS (S3…). L'utilisateur plakar a
# un UID fixe : les dossiers appdata lui appartiennent (chown 1000:1000).
RUN apk add --no-cache ca-certificates \
 && addgroup -g 1000 plakar \
 && adduser -D -u 1000 -G plakar -h /home/plakar plakar \
 && mkdir -p /home/plakar/.config/plakar /home/plakar/.cache/plakar /data /serve \
 && chown -R plakar:plakar /home/plakar /data /serve
COPY --from=plakar /out/plakar /usr/local/bin/plakar
# Copié depuis l'étage `contract` : l'image n'existe que si le test de contrat
# est passé.
COPY --from=contract /out/plakar-ui.contrat /usr/local/bin/plakar-ui
USER plakar
WORKDIR /home/plakar
# Le service n'écoute que sur localhost : seul le sidecar Tailscale, qui
# partage son réseau, le joint.
ENV PLAKAR_UI_ADDR=127.0.0.1:8080 \
    PLAKAR_UI_PLAKAR_BIN=/usr/local/bin/plakar \
    PLAKAR_UI_STATE=/data/plakar-ui-state.json \
    PLAKAR_UI_SERVE_CONFIG=/serve/serve.json
# Config plakar (stores.yml, passphrases : sensible), cache plakar, état.
VOLUME ["/home/plakar/.config/plakar", "/home/plakar/.cache/plakar", "/data"]
# Au premier lancement sur un cache neuf, plakar imprime sur sa sortie standard
# un avis sur son contrôle de mises à jour, qui fausserait la première sortie
# lue par le service. L'image tranche avant lui : contrôle désactivé (réglage
# idempotent, rangé dans le cache), les montées de version se font ici.
ENTRYPOINT ["/bin/sh", "-c", "plakar -disable-security-check >/dev/null && exec /usr/local/bin/plakar-ui"]
