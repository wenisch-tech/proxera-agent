#!/usr/bin/with-contenv bashio

# Required options
export PROXERA_SERVER_URL="$(bashio::config 'server_url')"
export PROXERA_API_KEY="$(bashio::config 'api_key')"
export PROXERA_LOG_LEVEL="$(bashio::config 'log_level')"

# Optional overrides — only set if the user provided a non-empty value
if bashio::config.has_value 'heartbeat_interval'; then
    export PROXERA_HEARTBEAT_INTERVAL="$(bashio::config 'heartbeat_interval')"
fi

if bashio::config.has_value 'heartbeat_timeout'; then
    export PROXERA_HEARTBEAT_TIMEOUT="$(bashio::config 'heartbeat_timeout')"
fi

if bashio::config.has_value 'reconnect_base'; then
    export PROXERA_RECONNECT_BASE="$(bashio::config 'reconnect_base')"
fi

if bashio::config.has_value 'reconnect_max'; then
    export PROXERA_RECONNECT_MAX="$(bashio::config 'reconnect_max')"
fi

if bashio::config.has_value 'request_timeout'; then
    export PROXERA_REQUEST_TIMEOUT="$(bashio::config 'request_timeout')"
fi

if bashio::config.has_value 'concurrency_limit' && [ "$(bashio::config 'concurrency_limit')" != "0" ]; then
    export PROXERA_CONCURRENCY_LIMIT="$(bashio::config 'concurrency_limit')"
fi

bashio::log.info "Starting proxera-agent..."
bashio::log.info "Server URL: ${PROXERA_SERVER_URL}"

exec /usr/bin/proxera-agent
