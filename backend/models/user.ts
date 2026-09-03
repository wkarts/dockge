import jwt from "jsonwebtoken";
import { R } from "redbean-node";
import { BeanModel } from "redbean-node/dist/bean-model";
import { generatePasswordHash, shake256, SHAKE256_LENGTH } from "../password-hash";

export class User extends BeanModel {
    /**
     * Reset a user password and invalidate every previously issued session.
     * @param {number} userID ID of user to update
     * @param {string} newPassword Users new password
     * @returns {Promise<string>} Newly persisted password hash
     */
    static async resetPassword(userID : number, newPassword : string): Promise<string> {
        const hash = generatePasswordHash(newPassword);
        await R.exec("UPDATE `user` SET password = ?, auth_revision = COALESCE(auth_revision, 1) + 1 WHERE id = ? ", [
            hash,
            userID
        ]);
        return hash;
    }

    /**
     * Reset this user's password and keep the in-memory bean synchronized
     * with the persisted hash/revision.
     * @param {string} newPassword
     * @returns {Promise<void>}
     */
    async resetPassword(newPassword : string) {
        this.password = await User.resetPassword(this.id, newPassword);
        this.auth_revision = Number(this.auth_revision || 1) + 1;
    }

    /**
     * Create a new JWT for a user. auth_revision intentionally participates
     * in the session contract so password/2FA changes invalidate old tokens.
     * @param {User} user The User to create a JsonWebToken for
     * @param {string} jwtSecret The key used to sign the JsonWebToken
     * @returns {string} the JsonWebToken as a string
     */
    static createJWT(user : User, jwtSecret : string) {
        return jwt.sign({
            username: user.username,
            h: shake256(user.password, SHAKE256_LENGTH),
            r: Number(user.auth_revision || 1),
        }, jwtSecret, {
            expiresIn: "7d",
        });
    }

}

export default User;
