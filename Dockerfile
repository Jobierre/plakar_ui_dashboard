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

# Base glibc, et non alpine (musl) : les plugins installés par `plakar pkg add`
# (s3…) sont des binaires publiés par Plakar, liés à la glibc. Sous musl, leur
# chargeur /lib64/ld-linux-x86-64.so.2 manque et plakar échoue sur un trompeur
# « fork/exec …/s3Storage: no such file or directory ».
FROM debian:bookworm-slim
# Certificats pour joindre les Stores en HTTPS (S3…). L'utilisateur plakar a
# un UID fixe : les dossiers appdata lui appartiennent (chown 1000:1000).
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd -g 1000 plakar \
 && useradd -m -u 1000 -g plakar -d /home/plakar -s /usr/sbin/nologin plakar \
 && mkdir -p /home/plakar/.config/plakar /home/plakar/.cache/plakar \
             /home/plakar/.local/share/plakar /data /serve \
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
# Config plakar (stores.yml, passphrases : sensible), cache plakar, paquets
# plakar (plugins de `plakar pkg add`, perdus sinon à chaque recréation), état.
VOLUME ["/home/plakar/.config/plakar", "/home/plakar/.cache/plakar", "/home/plakar/.local/share/plakar", "/data"]
# Au premier lancement sur un cache neuf, plakar imprime sur sa sortie standard
# un avis sur son contrôle de mises à jour, qui fausserait la première sortie
# lue par le service. L'image tranche avant lui : contrôle désactivé (réglage
# idempotent, rangé dans le cache), les montées de version se font ici.
ENTRYPOINT ["/bin/sh", "-c", "plakar -disable-security-check >/dev/null && exec /usr/local/bin/plakar-ui"]
