CREATE TABLE if NOT EXISTS malumotlarqizil (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    malumotlar_name varchar NOT NULL,
    malumotlar_value int NOT NULL,
    date DATE NOT NULL,
    timee TIME NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, 
    update_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, 
    deleted_at TIMESTAMP
);