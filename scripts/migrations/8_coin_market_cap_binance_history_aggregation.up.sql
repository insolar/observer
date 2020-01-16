-- CMC
create table coin_market_cap_stats_aggregate
(
    interval_time timestamp primary key,
    price_sum     numeric not null,
    count         int     not null
);


CREATE OR REPLACE FUNCTION update_coin_market_cap_stats_aggregate() RETURNS TRIGGER AS
$$
BEGIN
    insert into coin_market_cap_stats_aggregate (interval_time, price_sum, count)
        -- group date by 3 hour intervals 00 - 08, 08 - 16, 16 - 23.59
    values (new.created :: date + concat(floor(extract(hour from new.created) / 8) * 8, ':00:00') :: time, new.price, 1)
    on conflict (interval_time) do update set price_sum = coin_market_cap_stats_aggregate.price_sum + excluded.price_sum,
                                              count     = coin_market_cap_stats_aggregate.count + 1;
    return null;
END
$$ LANGUAGE 'plpgsql';

create trigger coin_market_cap_stats_aggregate_trigger
    after insert
    on coin_market_cap_stats
    for each row
execute procedure update_coin_market_cap_stats_aggregate();

-- For additional security and being sure about data in the stats table
CREATE OR REPLACE FUNCTION coin_market_cap_stats_insert_only() RETURNS TRIGGER AS
$$
BEGIN
    raise 'UPDATE / DELETE / TRUNCATE are not allowed on coin_market_cap_stats, because triggers are involved!';
END
$$ LANGUAGE 'plpgsql';

create trigger coin_market_cap_stats_ins_only
    before update or delete or truncate
    on coin_market_cap_stats
execute procedure coin_market_cap_stats_insert_only();


-- Binance
create table binance_stats_aggregate
(
    interval_time timestamp primary key,
    price_sum     numeric not null,
    count         int     not null
);

CREATE OR REPLACE FUNCTION update_binance_stats_aggregate() RETURNS TRIGGER AS
$$
BEGIN
    insert into binance_stats_aggregate (interval_time, price_sum, count)
        -- group date by 3 hour intervals 00 - 08, 08 - 16, 16 - 23.59
    values (new.created :: date + concat(floor(extract(hour from new.created) / 8) * 8, ':00:00') :: time, new.price, 1)
    on conflict (interval_time) do update set price_sum = binance_stats_aggregate.price_sum + excluded.price_sum,
                                              count     = binance_stats_aggregate.count + 1;
    return null;
END
$$ LANGUAGE 'plpgsql';

create trigger binance_stats_aggregate_trigger
    after insert
    on binance_stats
    for each row
execute procedure update_binance_stats_aggregate();

-- For additional security and being sure about data in the stats table
CREATE OR REPLACE FUNCTION binance_stats_insert_only() RETURNS TRIGGER AS
$$
BEGIN
    raise 'UPDATE / DELETE / TRUNCATE are not allowed on binance_stats, because triggers are involved!';
END
$$ LANGUAGE 'plpgsql';

create trigger binance_stats_ins_only
    before update or delete or truncate
    on binance_stats
execute procedure binance_stats_insert_only();
