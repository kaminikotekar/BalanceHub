-- Create a table for remote servers
CREATE TABLE servers (
    "pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
    "ipaddress" varchar(25),
    "port" varchar(10),
    "pathconstraint" boolean DEFAULT false,
    "ipconstraint" boolean DEFAULT false

);

-- create a table for storing url paths
CREATE TABLE pathmappings (
    "pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    "path" text,
    "serverid" integer NOT NULL,
    FOREIGN KEY(serverid) REFERENCES servers(pkid)
);

-- Create a table for storing ip constraints
CREATE TABLE addressmappings(
    "pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    "ipaddress" varchar(25),
    "serverid" integer NOT NULL,
    FOREIGN KEY(serverid) REFERENCES servers(pkid)
);