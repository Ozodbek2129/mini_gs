CREATE TABLE if NOT EXISTS gs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(100) NOT NULL,
    email VARCHAR(50) UNIQUE NOT NULL,
    active BOOLEAN DEFAULT false,
    image TEXT DEFAULT 'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, 
    update_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, 
    deleted_at TIMESTAMP
);