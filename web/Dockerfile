FROM node:24-alpine
WORKDIR /app
COPY package.json package-lock.json ./
COPY web/package.json ./web/package.json
RUN npm ci
COPY web ./web
RUN npm run build
EXPOSE 5173
CMD ["npm", "run", "dev"]
