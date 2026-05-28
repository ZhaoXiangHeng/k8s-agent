FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/node:24-alpine AS build
WORKDIR /src

# Keycloak OIDC config — K8s production defaults (Vite build-time vars)
ARG VITE_AUTH_MODE=keycloak
ARG VITE_KEYCLOAK_URL=/auth
ARG VITE_KEYCLOAK_REALM=k8s-ai
ARG VITE_KEYCLOAK_CLIENT_ID=k8s-ai-frontend
ARG VITE_DATA_MODE=api

ENV VITE_AUTH_MODE=${VITE_AUTH_MODE}
ENV VITE_KEYCLOAK_URL=${VITE_KEYCLOAK_URL}
ENV VITE_KEYCLOAK_REALM=${VITE_KEYCLOAK_REALM}
ENV VITE_KEYCLOAK_CLIENT_ID=${VITE_KEYCLOAK_CLIENT_ID}
ENV VITE_DATA_MODE=${VITE_DATA_MODE}

COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/nginx:1.27-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
