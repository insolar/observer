create table xns_coin_stats
(
    id          serial    not null
        constraint xns_coin_stats_pk
            primary key,
    created     timestamp not null,
    total       numeric(24),
    max         numeric(24),
    circulating numeric(24)
);