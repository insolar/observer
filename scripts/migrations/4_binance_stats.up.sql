create table binance_stats
(
    symbol text not null,
    symbol_price_btc text not null,
    symbol_price_usd text not null,
    btc_price_usd text not null,
    price_change_percent text not null,
    created timestamp not null
);

create index idx_binance_stats_created
    on binance_stats (created);
