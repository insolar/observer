create table coin_market_cap_stats
(
    price numeric not null,
    percent_change_24_hours numeric not null,
    rank int not null,
    market_cap numeric not null,
    volume_24_hours numeric not null,
    circulating_supply numeric  not null,
    created timestamp not null
);

create index idx_coin_market_cap_stats_created
    on coin_market_cap_stats (created);
