create table if not exists burned_balance
(
    id bigserial not null primary key,
    balance varchar(256),
    account_state bytea
);
