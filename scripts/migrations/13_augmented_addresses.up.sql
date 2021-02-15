create table if not exists augmented_addresses
(
    member_ref bytea not null
        constraint augmented_addresses_pkey
            primary key,
    address varchar(256) not null
);
