set search_path = yabatasg;

drop table if exists bus_route;
drop table if exists bus_service;
drop table if exists bus_stop;

CREATE TABLE IF NOT EXISTS bus_stop (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bus_stop_code VARCHAR(5) UNIQUE NOT NULL,
    road_name VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    location GEOMETRY(POINT, 4326) NOT NULL
);

CREATE INDEX if not exists idx_bus_stop_location ON bus_stop USING GIST (location);

CREATE TABLE IF NOT EXISTS bus_service (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_number VARCHAR(10) NOT NULL,
    operator VARCHAR(10) NOT NULL,
    direction SMALLINT NOT NULL,
    category VARCHAR(20) NOT NULL,
    origin_bus_stop_code VARCHAR(5) NOT NULL,
    destination_bus_stop_code VARCHAR(5) NOT NULL,
    am_peak_freq_min INT NOT NULL,
    am_peak_freq_max INT NOT NULL,
    am_offpeak_freq_min INT NOT NULL,
    am_offpeak_freq_max INT NOT NULL,
    pm_peak_freq_min INT NOT NULL,
    pm_peak_freq_max INT NOT NULL,
    pm_offpeak_freq_min INT NOT NULL,
    pm_offpeak_freq_max INT NOT NULL,
    loop_description TEXT NOT NULL,
    CONSTRAINT uq_service_direction UNIQUE (service_number, direction),
    CONSTRAINT fk_origin_stop
        FOREIGN KEY (origin_bus_stop_code)
        REFERENCES bus_stop (bus_stop_code)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT fk_destination_stop
        FOREIGN KEY (destination_bus_stop_code)
        REFERENCES bus_stop (bus_stop_code)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS bus_route (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_number VARCHAR(10) NOT NULL,
    direction SMALLINT NOT NULL,
    stop_sequence INT NOT NULL,
    bus_stop_code VARCHAR(5) NOT NULL,
    distance DECIMAL(8, 2) NOT NULL,
    weekday_first_bus TIME NOT NULL,
    weekday_last_bus TIME NOT NULL,
    sat_first_bus TIME NOT NULL,
    sat_last_bus TIME NOT NULL,
    sun_first_bus TIME NOT NULL,
    sun_last_bus TIME NOT NULL,
    CONSTRAINT fk_bus_stop
        FOREIGN KEY (bus_stop_code)
        REFERENCES bus_stop (bus_stop_code)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT fk_bus_service
        FOREIGN KEY (service_number, direction)
        REFERENCES bus_service (service_number, direction)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT uq_bus_route UNIQUE (service_number, direction, stop_sequence, bus_stop_code)
);

COMMENT ON COLUMN bus_route.distance IS 'Distance in kilometres';
