import { FOUNDED_YEAR } from '../data/site.ts'
import { useContent } from '../data/content.ts'
import { Link } from 'react-router-dom'
import { IconFacebook, IconXSocial, IconInstagram, IconLinkedin, IconYoutube, IconPhone, IconMail, IconPin } from './icons.tsx'
import { Logo } from './ui.tsx'

/** The group's real channels, supplied by the client (change order item 15). */
const SOCIAL_LINKS = [
  { label: 'Facebook', href: 'https://www.facebook.com/quadisgroupofhotelss', Icon: IconFacebook },
  { label: 'X', href: 'https://x.com/quadis_hotels', Icon: IconXSocial },
  { label: 'Instagram', href: 'https://www.instagram.com/quadis_groupofhotels/', Icon: IconInstagram },
  { label: 'LinkedIn', href: 'https://www.linkedin.com/company/quadis-group-of-hotels', Icon: IconLinkedin },
  { label: 'YouTube', href: 'https://www.youtube.com/@QuadisGroupofHotels', Icon: IconYoutube },
]

interface FooterLink { label: string; to: string }
/*
 * One entry per brand, not per property — five names covering nine hotels.
 * Amaltas was added on 1 Sep 2026 because it is a sixth brand: every other
 * property in the group is reachable through a name already on this list, and
 * that one is not.
 */
const HOTELS_LINKS: FooterLink[] = [
  { label: 'Hotel Amaltas International', to: '/hotels/hotel-amaltas-international' },
  { label: 'Hotel Amar Inn', to: '/hotels/hotel-amar-inn' },
  { label: 'Hotel Amby Inn', to: '/hotels/hotel-amby-inn-lajpat-nagar-ii' },
  { label: 'Hotel Downtown', to: '/hotels/hotel-downtown-east-of-kailash' },
  { label: 'Hotel Cladis', to: '/hotels/hotel-cladis-sector-15-noida' },
  { label: 'Hotel Quadis', to: '/hotels/hotel-quadis-sector-51-noida' },
]
/*
 * Mirrors the "Important Links" block on the client's existing site, minus
 * Career and Blog, which have no page here. Privacy and Terms are not optional
 * furniture: Razorpay requires both to be reachable from the site before it
 * will approve a merchant account, and a page nothing links to is one a
 * crawler will not find after the cutover.
 */
const IMPORTANT: FooterLink[] = [
  { label: 'Contact Us', to: '/contact' },
  { label: 'Privacy Policy', to: '/privacy-policy' },
  { label: 'Terms & Conditions', to: '/terms-and-conditions' },
  { label: 'Cancellation Policy', to: '/cancellation-policy' },
]

export default function Footer() {
  const { t } = useContent()
  return (
    <footer className="footer bg-darkest">
      <div className="container footer__inner">
        <div className="footer__col footer__brand">
          <Logo variant="footer" />
          <p className="footer__blurb">{t('footer.tagline')}</p>
        </div>

        <nav className="footer__col" aria-label="Hotels">
          <h4 className="footer__head">HOTELS</h4>
          <ul className="footer__links">
            {HOTELS_LINKS.map((l) => (<li key={l.label}><Link to={l.to}>{l.label}</Link></li>))}
          </ul>
        </nav>

        <nav className="footer__col" aria-label="Important links">
          <h4 className="footer__head">IMPORTANT LINKS</h4>
          <ul className="footer__links">
            {IMPORTANT.map((l) => (<li key={l.label}><Link to={l.to}>{l.label}</Link></li>))}
          </ul>
        </nav>

        <div className="footer__col">
          <h4 className="footer__head">CONNECT WITH US</h4>
          <ul className="footer__links footer__contact">
            <li><a href="tel:+919217373532"><IconPhone /> <span>+91 92173 73532</span></a></li>
            <li><a href="mailto:info@quadishotels.com"><IconMail /> <span>info@quadishotels.com</span></a></li>
            <li className="footer__addr"><IconPin width={18} height={18} /> <span>H-22, LT SH Jagpal Singh, Sector-51, Noida, Gautam Buddha Nagar, UP 201307</span></li>
          </ul>
        </div>
      </div>

      <div className="container footer__bottom">
        <div className="footer__social" aria-label="Social links">
          {SOCIAL_LINKS.map(({ label, href, Icon }) => (
            <a key={label} href={href} aria-label={label} target="_blank" rel="noopener noreferrer">
              <Icon />
            </a>
          ))}
        </div>
        <p className="footer__copy">© {FOUNDED_YEAR}–{new Date().getFullYear()} Quadis Services Private Limited. All Rights Reserved.</p>
      </div>
    </footer>
  )
}
