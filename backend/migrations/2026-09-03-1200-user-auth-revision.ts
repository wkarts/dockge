import { Knex } from "knex";

export async function up(knex: Knex): Promise<void> {
    const exists = await knex.schema.hasColumn("user", "auth_revision");
    if (!exists) {
        await knex.schema.alterTable("user", (table) => {
            table.integer("auth_revision").notNullable().defaultTo(1);
        });
    }
}

export async function down(knex: Knex): Promise<void> {
    const exists = await knex.schema.hasColumn("user", "auth_revision");
    if (exists) {
        await knex.schema.alterTable("user", (table) => {
            table.dropColumn("auth_revision");
        });
    }
}
